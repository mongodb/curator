package sthree

import (
	"fmt"
	"io/ioutil"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/queue"
	"github.com/tychoish/grip"
)

func init() {
	// adds, at process startup time.

	grip.CatchError(mime.AddExtensionType(".deb", "application/octet-stream"))
	grip.CatchError(mime.AddExtensionType(".gz", "application/x-gzip"))
	grip.CatchError(mime.AddExtensionType(".json", "application/json"))
	grip.CatchError(mime.AddExtensionType(".rpm", "application/x-redhat-package-manager"))
	grip.CatchError(mime.AddExtensionType(".tgz", "application/x-gzip"))
	grip.CatchError(mime.AddExtensionType(".txt", "text/plain"))
	grip.CatchError(mime.AddExtensionType(".yaml", "text/x-yaml"))
	grip.CatchError(mime.AddExtensionType(".yml", "text/x-yaml"))
}

// AWSConnectionConfiguration defines configuration, including
// authentication credentials and AWS region, used when creating new
// connections to AWS components.
type AWSConnectionConfiguration struct {
	// AWS auth credentials, using a type defined in the goamz
	// package.
	Auth aws.Auth

	// Specify a region to use in the AWS connection. For S3
	// operations this should not matter.
	Region aws.Region
}

// Bucket defines a tracking object for a bucket. Create access using the
// global GetBucket factory, which allows users to pool bucket operations.
type Bucket struct {
	// The permission defined by NewFilePermission is used for all
	// Put operations in the bucket.
	NewFilePermission s3.ACL
	open              bool
	credentials       AWSConnectionConfiguration
	bucket            *s3.Bucket
	s3                *s3.S3
	registry          *bucketRegistry
	name              string
	numJobs           int
	queue             amboy.Queue
}

// NewBucket clones the settings of one bucket into a new bucket. The
// new bucket is registered, and can be reused by other callers before
// it is closed.
func (b *Bucket) NewBucket(name string) *Bucket {
	new := &Bucket{
		name:              name,
		NewFilePermission: b.NewFilePermission,
		credentials:       b.credentials,
		numJobs:           b.numJobs,
	}

	b.registry.registerBucket(new)
	return new
}

func (b *Bucket) String() string {
	return b.name
}

// SetCredentials allows you to override the configured credentials in
// the Bucket instance. Bucket instances have default credentials
// picked from either the AWS_ACCESS_KEY_ID and AWS_ACCESS_KEY
// environment variables or, if they are not defined then from the
// "$HOME/.aws/credentials" file (using the "default" profile unless
// the AWS_PROFILE environment variable is set.)
//
// This method redefines the internal representation of the
// credentials and connection to S3. Callers can use this function
// after the connection is open, *but* this may affect in progress
// jobs in undefined ways.
func (b *Bucket) SetCredentials(c AWSConnectionConfiguration) {
	b.credentials = c
	b.s3 = s3.New(b.credentials.Auth, b.credentials.Region)
	b.bucket = b.s3.Bucket(b.name)
}

// SetNumJobs allows callers to change the number of worker threads
// the Bucket will start to process sync jobs. Returns an error if the
// Bucket is open and has a running queue.
func (b *Bucket) SetNumJobs(n int) error {
	if b.open {
		return fmt.Errorf("numJobs=%d, cannot change for a running queue", b.numJobs)
	}

	b.numJobs = n
	return nil
}

// Open creates connections to S3 and starts a the worker pool to
// process sync to/from jobs. Returns an error if there are issues
// creating creating the worker queue. Does *not* return an error if
// the Bucket has been opened, and is a noop in this case.
func (b *Bucket) Open() error {
	if b.open {
		return nil
	}

	if b.s3 == nil {
		b.s3 = s3.New(b.credentials.Auth, b.credentials.Region)
	}

	if b.bucket == nil || b.bucket.Name != b.name {
		b.bucket = b.s3.Bucket(b.name)
	}

	b.open = true

	b.queue = queue.NewLocalUnordered(b.numJobs)

	return b.queue.Start()
}

// Close waits for all pending jobs to return and then releases all
// worker resources and releases the object from the internal registry
// of buckets. Close is a noop if the bucket is not open.
func (b *Bucket) Close() {
	defer b.registry.removeBucket(b)

	if b.open {
		b.queue.Close()
		b.open = false
	}
}

type s3Item struct {
	Name string
	MD5  string
}

// list returns a channel of strings of key names in the bucket. Allows
// you to specify a prefix key that will limit the results returned in
// the channel. If you do not want to limit using a prefix, pass an
// empty string as the sole argument for list().
func (b *Bucket) list(prefix string) <-chan *s3Item {
	output := make(chan *s3Item, 100)

	// if the prefix doesn't have a trailing slash and isn't the
	// empty string, then we can have weird effects with files that
	// have the same prefix.
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	go func() {
		var lastKey string
		for {
			results, err := b.bucket.List(prefix, "", lastKey, 1000)
			if err != nil {
				grip.Error(err)
				break
			}

			for _, key := range results.Contents {
				lastKey = key.Key

				output <- &s3Item{
					Name: key.Key,
					MD5:  strings.Trim(key.ETag, "\" "),
				}
			}
			if !results.IsTruncated {
				break
			}
		}
		close(output)
	}()

	return output
}

// contents wraps and operates as list, but returns a map of names to
// s3Item objects for random access patterns.
func (b *Bucket) contents(prefix string) map[string]*s3Item {
	output := make(map[string]*s3Item)
	for file := range b.list(prefix) {
		output[file.Name] = file
	}

	return output
}

// Put uploads the local fileName to the remote path object in the
// current bucket. Put attempts to determine the content type based on
// the extension of the file, and defaults to "text/plain" if the
// extension is not known. The permissions on the object use the value
// of the Bucket.NewFilePermission property. Returns an error if the
// underlying Put operation returns an error.
func (b *Bucket) Put(fileName, path string) error {
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return fmt.Errorf("file '%s' does not exist", fileName)
	}

	mimeType := getMimeType(fileName)
	contents, err := ioutil.ReadFile(fileName)

	if err != nil {
		return err
	}

	grip.Debugf("uploading %s -> (%s) %s", fileName, b.name, path)
	return b.bucket.Put(path, contents, mimeType, b.NewFilePermission, s3.Options{})
}

// getMimeType takes a file name, attempts to determine the extension
// and resolve a MIME type for this value. If there is no resolvable
// MIME type, getMimeType returns "text/plain". This is only used in
// Put(), but is separate so that it can be tested.
func getMimeType(fileName string) string {
	parts := strings.Split(fileName, ".")

	mimeType := mime.TypeByExtension(fmt.Sprintf(".%s", parts[len(parts)-1]))
	if mimeType == "" {
		return "text/plain"
	}

	return mimeType
}

// Get writes the content of the S3 object located at "path" to the
// local file at the "fileName", creating enclosing directories as
// needed.
func (b *Bucket) Get(path, fileName string) error {
	grip.Debugf("downloading (%s) %s -> %s", b.name, path, fileName)

	data, err := b.bucket.Get(path)
	if err != nil {
		return err
	}

	dirName := filepath.Dir(fileName)
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		err = os.MkdirAll(dirName, 0755)
		if err != nil {
			return err
		}
		grip.Debugf("created directory '%s' for object %s", dirName, fileName)
	}

	return ioutil.WriteFile(fileName, data, 0644)
}

// Delete removes a single object from an S3 bucket.
func (b *Bucket) Delete(path string) error {
	grip.Noticef("removing %s.%s", b.name, path)
	return b.bucket.Del(path)
}

// DeleteMany takes a variable number of strings, and sends a single
// request to S3 to delete those keys from the bucket.
func (b *Bucket) DeleteMany(paths ...string) error {
	if len(paths) == 1 {
		// getting the bucket contents is a comparatively
		// expensive operation so makes sense to avoid it in
		// this case.
		return b.Delete(paths[0])
	}

	catcher := grip.NewCatcher()

	toDelete := s3.Delete{}
	contents := b.contents("")
	count := 0
	for _, p := range paths {
		key, ok := contents[p]
		if !ok {
			grip.Warningf("path %s does not exist in bucket %s", p, b.name)
			continue
		}

		// DeleteMulti maxes out at 1000 items per request. We
		// should batch accordingly too.
		if count == 1000 {
			grip.Debugf("sending a batch of delete operations to %s", b.name)
			catcher.Add(b.bucket.DelMulti(toDelete))

			// reset the counters
			toDelete = s3.Delete{}
			count = 0
		}

		grip.Noticef("removing group, with %s.%s", b.name, key)

		count++
		toDelete.Objects = append(toDelete.Objects, s3.Object{Key: key.Name})
	}

	if len(toDelete.Objects) > 0 {
		grip.Debugf("sending last batch of delete operations to %s", b.name)
		catcher.Add(b.bucket.DelMulti(toDelete))
	}

	return catcher.Resolve()
}

// DeletePrefix removes all items in a bucket that have key names that
// begin with a specific prefix.
func (b *Bucket) DeletePrefix(prefix string) error {
	catcher := grip.NewCatcher()

	toDelete := s3.Delete{}
	items := b.list(prefix)
	count := 0

	for {
		// DeleteMulti maxes out at 1000 items per request. We
		// should batch accordingly too.
		if count == 1000 {
			grip.Debugf("sending a batch of delete operations to %s", b.name)
			catcher.Add(b.bucket.DelMulti(toDelete))

			// reset the counters
			toDelete = s3.Delete{}
			count = 0
		}

		// pull from a channel, and add to the batch.
		key, ok := <-items
		if ok {
			count++

			toDelete.Objects = append(toDelete.Objects, s3.Object{Key: key.Name})
			grip.Noticef("removing group, with %s.%s", b.name, key.Name)

			continue
		}

		break
	}

	if len(toDelete.Objects) > 0 {
		grip.Debugf("sending last batch of delete operations to %s", b.name)
		catcher.Add(b.bucket.DelMulti(toDelete))
	}

	return catcher.Resolve()
}

// SyncTo takes a local path, typically directory, and an S3 path
// prefix, and dispatches a job to upload that file to S3 if it does
// not exist or if the local file has different content from the
// remote file. All operations execute in the worker pool, and SyncTo
// waits for all jobs to complete before returning an aggregated error.
func (b *Bucket) SyncTo(local, prefix string) error {
	grip.Infof("sync push %s -> %s/%s", local, b.name, prefix)

	catcher := grip.NewCatcher()
	remote := b.contents(prefix)

	var counter int
	catcher.Add(filepath.Walk(local, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			grip.Critical(err)
			return err
		}

		if info.IsDir() {
			return nil
		}

		remoteFile, ok := remote[path]
		if !ok {
			remoteFile = &s3Item{Name: filepath.Join(prefix, path[len(local):])}
		}

		job := newSyncToJob(path, remoteFile, b)

		counter++

		return b.queue.Put(job)
	}))

	b.queue.Wait()
	catcher.Add(b.queue.Runner().Error())

	if catcher.HasErrors() {
		grip.Alertf("problem with sync push operation (%s -> %s/%s) [%d items]",
			local, b.name, prefix, counter)
	} else {
		grip.Infof("completed push operation. uploaded %d items to %s/%s", counter, b.name, prefix)
	}

	return catcher.Resolve()
}

// SyncFrom takes a local path and the prefix of a keyname in S3, and
// and downloads all objects in the bucket that have that prefix to
// the local system at the path specified by "local". Will *not*
// download files if the content of the local file have *not* changed.
// All operations execute in the worker pool, and SyncTo waits for all
// jobs to complete before returning an aggregated erro
func (b *Bucket) SyncFrom(local, prefix string) error {
	catcher := grip.NewCatcher()
	grip.Infof("sync pull %s/%s -> %s", b.name, prefix, local)

	for remote := range b.list(prefix) {
		job := newSyncFromJob(filepath.Join(local, remote.Name[len(prefix):]), remote, b)

		// add the job to the queue
		catcher.Add(b.queue.Put(job))
	}

	b.queue.Wait()
	catcher.Add(b.queue.Runner().Error())

	if catcher.HasErrors() {
		grip.Alertf("problem with sync pull operation (%s/%s -> %s)",
			b.name, prefix, local)
	} else {
		grip.Infof("completed pull operation from %s/%s -> %s", b.name, prefix, local)
	}

	return catcher.Resolve()
}
