package sthree

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	"github.com/jpillora/backoff"
	"github.com/mongodb/amboy"
	"github.com/mongodb/amboy/queue"
	"github.com/pkg/errors"
	"github.com/mongodb/grip"
	"golang.org/x/net/context"
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

	// for jitter in expoential backoff
	rand.Seed(time.Now().Unix())
}

func getBackoff() *backoff.Backoff {
	return &backoff.Backoff{
		Min:    100 * time.Millisecond,
		Max:    5 * time.Second,
		Factor: 2,
		Jitter: true,
	}
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
	dryRun            bool
	credentials       AWSConnectionConfiguration
	bucket            *s3.Bucket
	s3                *s3.S3
	name              string
	numJobs           int
	numRetries        int
	queue             amboy.Queue
	mutex             sync.Mutex
	closer            context.CancelFunc
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
		numRetries:        20,
	}

	buckets.registerBucket(new)
	return new
}

// DryRunClone creates a copy of the bucket, which is not a shared
// resource, that runs with dry-run mode. In this mode, all PUT
// and DELETE operations are no-ops, with more logging, but all other
// operations (including "GET" operations) are as normal.
func (b *Bucket) DryRunClone() (*Bucket, error) {
	clone, err := b.Clone()

	if err != nil {
		return nil, err
	}
	clone.dryRun = true

	if b.queue != nil {
		err = clone.Open()
		if err != nil {
			return nil, err
		}
	}

	return clone, nil
}

// Clone returns a copy of the existing bucket, which is not a shared
// resource.
func (b *Bucket) Clone() (*Bucket, error) {
	clone := &Bucket{
		name:              b.name,
		NewFilePermission: b.NewFilePermission,
		credentials:       b.credentials,
		numJobs:           b.numJobs,
		numRetries:        b.numRetries,
	}

	if b.queue != nil {
		err := clone.Open()
		if err != nil {
			return nil, err
		}
	}

	return clone, nil
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
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.queue != nil {
		return errors.Errorf("numJobs=%d, cannot change for a running queue", b.numJobs)
	}

	b.numJobs = n
	return nil
}

// SetNumRetries allows callers to change the number of retries put
// and get operations will take in the cse of an error.
func (b *Bucket) SetNumRetries(n int) error {
	if n <= 0 {
		return errors.Errorf("numRetries=%d, must be larger than 0", n)
	}

	b.numRetries = n
	return nil
}

// Open creates connections to S3 and starts a the worker pool to
// process sync to/from jobs. Returns an error if there are issues
// creating creating the worker queue. Does *not* return an error if
// the Bucket has been opened, and is a noop in this case.
func (b *Bucket) Open() error {
	if b.s3 == nil {
		b.s3 = s3.New(b.credentials.Auth, b.credentials.Region)
	}

	if b.bucket == nil || b.bucket.Name != b.name {
		b.bucket = b.s3.Bucket(b.name)
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.queue == nil {
		ctx, cancel := context.WithCancel(context.TODO())
		b.closer = cancel

		b.queue = queue.NewLocalUnordered(b.numJobs)
		return errors.Wrap(b.queue.Start(ctx), "starting worker queue for sync jobs")
	}

	return nil
}

// Close waits for all pending jobs to return and then releases all
// worker resources and releases the object from the internal registry
// of buckets. Close is a noop if the bucket is not open.
func (b *Bucket) Close() {
	defer buckets.removeBucket(b)

	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.queue != nil {
		b.queue.Runner().Close()
		b.queue = nil
	}

	if b.closer != nil {
		b.closer()
	}
}

// IsOpen returns true if the queue, used by the sync operations is
// initialized, which is the final stage of the Open operation.
func (b *Bucket) IsOpen() bool {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	return b.queue != nil
}

// list returns a channel of strings of key names in the bucket. Allows
// you to specify a prefix key that will limit the results returned in
// the channel. If you do not want to limit using a prefix, pass an
// empty string as the sole argument for list().
func (b *Bucket) list(prefix string) <-chan s3.Key {
	output := make(chan s3.Key, 100)

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

				output <- key
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
func (b *Bucket) contents(prefix string) map[string]s3.Key {
	output := make(map[string]s3.Key)

	for file := range b.list(prefix) {
		output[file.Key] = file
	}

	return output
}

// Exists checks to see if a key exists in the bucket, retrying the request, if needed.
func (b *Bucket) Exists(path string) (bool, error) {
	var exists bool
	var err error

	backoff := getBackoff()

	for i := 1; i <= b.numRetries; i++ {
		exists, err = b.bucket.Exists(path)
		if err == nil {
			return exists, nil
		}

		err = errors.Wrapf(err, "error s3.EXISTS for %s/%s on attempt %d", b.name, path, i)

		if i < b.numRetries {
			dur := backoff.Duration()
			grip.Debugf("retrying s3.EXISTS attempt %d of %d (after %s), for %s (%+v)",
				i, b.numRetries, dur, path, err)
			time.Sleep(dur)
		}
	}

	return exists, err
}

// Put uploads the local fileName to the remote path object in the
// current bucket. Put attempts to determine the content type based on
// the extension of the file, and defaults to "text/plain" if the
// extension is not known. The permissions on the object use the value
// of the Bucket.NewFilePermission property. Returns an error if the
// underlying Put operation returns an error.
func (b *Bucket) Put(fileName, path string) error {
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return errors.Errorf("file '%s' does not exist", fileName)
	}

	mimeType := getMimeType(fileName)
	contents, err := ioutil.ReadFile(fileName)

	if err != nil {
		return errors.Wrapf(err, "error reading file '%s' before s3.Put", fileName)
	}

	if b.dryRun {
		grip.Noticef("dry-run: would have uploaded %s -> %s/%s", fileName, b.name, path)
		return nil
	}

	return errors.Wrap(b.Write(contents, path, mimeType), "problem writing data to s3")
}

// Write allows you to write a byte slice of data to a specified key
// name (path) in s3 with the specified mimeType. The permissions on
// the object use the value of the Bucket.NewFilePermission
// property. Returns an error if the underlying Put operation returns
// an error.
//
// If you do not specify a mime type, sthree attempts to use the name
// of the key, and defaults to "text/plain".
func (b *Bucket) Write(data []byte, keyName, mime string) error {
	if b.dryRun {
		grip.Noticef("dry-run: would have written %d bytes to %s/%s", len(data), b.name, keyName)
		return nil
	}

	if mime == "" {
		mime = getMimeType(keyName)
	}

	catcher := grip.NewCatcher()
	backoff := getBackoff()
	for i := 1; i <= b.numRetries; i++ {
		err := b.bucket.Put(keyName, data, mime, b.NewFilePermission, s3.Options{})
		if err == nil {
			grip.Debugf("wrote %d bytes to %s/%s", len(data), b.name, keyName)
			return nil
		}

		catcher.Add(errors.Wrapf(err, "s3.PUT for key=%s failed on attempt %d", keyName, i))

		if i < b.numRetries {
			grip.Warningln(err, "retrying...")
			time.Sleep(backoff.Duration())
			grip.Debugf("retrying s3.PUT %d of %d, for %s", i, b.numRetries, keyName)
		}
	}

	if catcher.HasErrors() {
		return errors.Errorf("could not write %d bytes to %s/%s in %d attempts. Errors: %s",
			len(data), b.name, keyName, b.numRetries, catcher.Resolve())
	}

	return nil
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
	// do get in a retry loop:

	data, err := b.Read(path)
	if err != nil {
		return errors.Wrap(err, "problem reading data from s3")
	}

	dirName := filepath.Dir(fileName)
	if _, err = os.Stat(dirName); os.IsNotExist(err) {
		err = os.MkdirAll(dirName, 0755)
		if err != nil {
			return errors.Wrap(err, "creating directory for s3.Get operations")
		}
		grip.Debugf("created directory '%s' for object %s", dirName, fileName)
	}

	return errors.Wrapf(ioutil.WriteFile(fileName, data, 0644),
		"writing file %s during s3 get", fileName)
}

// Read returns the contents of a key as a byte array.
func (b *Bucket) Read(keyName string) ([]byte, error) {
	catcher := grip.NewCatcher()

	var data []byte
	var err error

	backoff := getBackoff()
	for i := 1; i <= b.numRetries; i++ {
		data, err = b.bucket.Get(keyName)

		if err == nil {
			grip.Debugf("downloaded %s/%s", b.name, keyName)
			catcher = grip.NewCatcher() // reset the error handler in the case of success
			break
		}

		catcher.Add(errors.Wrap(err, "aws error from s3.Get"))

		if i < b.numRetries {
			grip.Warningln(err, "retrying...")
			time.Sleep(backoff.Duration())
			grip.Debugf("retrying s3.GET %d of %d, for %s", i, b.numRetries, keyName)
		}
	}

	// return early if we encountered an error attempting to read the data
	if catcher.HasErrors() {
		return []byte{}, errors.Errorf("could not download %s/%s in %d attempts. Errors: %s",
			b.name, keyName, b.numRetries, catcher.Resolve())
	}

	return data, nil
}

// Delete removes a single object from an S3 bucket.
func (b *Bucket) Delete(path string) error {
	grip.Noticef("removing %s.%s", b.name, path)

	if b.dryRun {
		grip.Infof("dry-run: would delete %s from %s", path, b.name)
		return nil
	}

	return errors.Wrapf(b.bucket.Del(path), "deleting %s from %s", path, b.name)
}

// DeleteMany takes a variable number of strings, and sends a single
// request to S3 to delete those keys from the bucket.
func (b *Bucket) DeleteMany(paths ...string) error {
	if len(paths) == 1 {
		// getting the bucket contents is a comparatively
		// expensive operation so makes sense to avoid it in
		// this case.
		if b.dryRun {
			grip.Infof("dry-run: deleting object %s as a single object in a multi-delete call", paths[0])
			return nil
		}
		return errors.Wrap(b.Delete(paths[0]), "single delete operation in multi-delete call")
	}

	contents := b.contents("")
	toDelete := make(chan s3.Key)
	go func() {
		for _, p := range paths {
			key, ok := contents[p]
			if !ok {
				grip.Warningf("path %s does not exist in bucket %s", p, b.name)
				continue
			}

			toDelete <- key
		}

		close(toDelete)
	}()

	return b.deleteGroup(toDelete)
}

// DeletePrefix removes all items in a bucket that have key names that
// begin with a specific prefix.
func (b *Bucket) DeletePrefix(prefix string) error {
	return b.deleteGroup(b.list(prefix))
}

// DeleteMatching removes all objects from a bucket, given a prefix,
// that match a regular expression.
func (b *Bucket) DeleteMatching(prefix, expression string) error {
	matcher, err := regexp.Compile(expression)
	if err != nil {
		return errors.Wrapf(err,
			"problem compiling expression %s for delete matching operation on %s",
			expression, b.name)
	}

	toDelete := make(chan s3.Key)

	go func() {
		var count int
		list := b.list(prefix)

		for item := range list {
			name := item.Key

			if matcher.MatchString(name) {
				toDelete <- item
				count++

				grip.NoticeWhenf(b.dryRun, "found %s/%s to delete", b.name, name)
				grip.DebugWhenf(!b.dryRun, "found %s/%s to delete", b.name, name)
			} else {
				grip.DebugWhenf(b.dryRun, "%s/%s does not match %s", b.name, name, expression)
			}
		}

		grip.Noticef("found %d files matching '%s' in '%s/%s' for deletion",
			count, expression, b.name, prefix)
		close(toDelete)
	}()

	return b.deleteGroup(toDelete)
}

func (b *Bucket) deleteGroup(items <-chan s3.Key) error {
	toDelete := s3.Delete{}
	count := 0
	catcher := grip.NewCatcher()

	for {
		// DeleteMulti maxes out at 1000 items per request. We
		// should batch accordingly too.
		if count == 1000 {
			if b.dryRun {
				grip.Infof("dry-run: would send a batch of delete operations to %s", b.name)
			} else {
				grip.Debugf("sending a batch of delete operations to %s", b.name)

				catcher.Add(errors.Wrapf(b.bucket.DelMulti(toDelete),
					"intermediate delete from %s, %d items encountered error",
					b.name, count))
			}

			// reset the counters
			toDelete = s3.Delete{}
			count = 0
		}

		// pull from a channel, and add to the batch.
		key, ok := <-items
		if ok {
			count++

			toDelete.Objects = append(toDelete.Objects, s3.Object{Key: key.Key})
			grip.Infof("removing group, with %s/%s", b.name, key.Key)

			continue
		}

		break
	}

	if len(toDelete.Objects) > 0 {
		if b.dryRun {
			grip.Infof("dry-run: would send last batch of delete operations to %s", b.name)
		} else {
			grip.Debugf("sending last batch of delete operations to %s", b.name)

			catcher.Add(errors.Wrapf(b.bucket.DelMulti(toDelete),
				"delete from %s, %d items encountered error",
				b.name, len(toDelete.Objects)))
		}
	}

	if catcher.HasErrors() {
		return catcher.Resolve()
	}

	grip.NoticeWhenf(b.dryRun, "dry-run: completed delete operation, removing %d items", count)
	grip.NoticeWhenf(!b.dryRun, "completed delete operation, removing %d items", count)
	return nil
}

// SyncTo takes a local path, typically directory, and an S3 path
// prefix, and dispatches a job to upload that file to S3 if it does
// not exist or if the local file has different content from the
// remote file. All operations execute in the worker pool, and SyncTo
// waits for all jobs to complete before returning an aggregated error.
func (b *Bucket) SyncTo(local, prefix string, withDelete bool) error {
	grip.Infof("sync push %s -> %s/%s", local, b.name, prefix)

	remote := b.contents(prefix)

	var counter int
	catcher := grip.NewCatcher()

	catcher.Add(filepath.Walk(local, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			grip.Critical(errors.Wrapf(err, "problem finding file %s", path))
			return nil
		}

		if info.IsDir() {
			return nil
		}

		var keyName string
		if local == path {
			keyName = filepath.Join(prefix, path)
		} else {
			// need the extra character to avoid missing this because of the leading slash.
			keyName = filepath.Join(prefix, path[len(local)+1:])
		}

		remoteFile, ok := remote[keyName]
		if !ok {
			remoteFile = s3.Key{Key: keyName}
		}

		job := newSyncToJob(b, path, remoteFile, withDelete)

		err = errors.Wrap(b.queue.Put(job), "problem putting syncTo job into queue")
		if err != nil {
			catcher.Add(err)
			return nil
		}

		counter++

		return nil
	}))

	amboy.WaitInterval(b.queue, 250*time.Millisecond)

	if catcher.HasErrors() {
		grip.Alertf("encountered %d problems putting jobs into the syncTo queue", catcher.Len())
		return catcher.Resolve()
	}

	if err := amboy.ResolveErrors(context.TODO(), b.queue); err != nil {
		m := fmt.Sprintf("problem with syncTo operation (%s -> %s/%s)",
			local, b.name, prefix)
		grip.Alert(m)
		return errors.Wrap(err, m)
	}

	grip.Infof("completed push operation. upload items to %s/%s [considered %d items]",
		b.name, prefix, counter)

	return nil
}

// SyncFrom takes a local path and the prefix of a keyname in S3, and
// and downloads all objects in the bucket that have that prefix to
// the local system at the path specified by "local". Will *not*
// download files if the content of the local file have *not* changed.
// All operations execute in the worker pool, and SyncTo waits for all
// jobs to complete before returning an aggregated erro
func (b *Bucket) SyncFrom(local, prefix string, withDelete bool) error {
	grip.Infof("sync pull %s/%s -> %s", b.name, prefix, local)

	for remote := range b.list(prefix) {
		job := newSyncFromJob(b, filepath.Join(local, remote.Key[len(prefix):]), remote, withDelete)

		// add the job to the queue
		if err := b.queue.Put(job); err != nil {
			return errors.Wrap(err, "problem putting syncFrom job into worker queue")
		}
	}

	amboy.WaitInterval(b.queue, 250*time.Millisecond)

	if err := amboy.ResolveErrors(context.TODO(), b.queue); err != nil {
		m := fmt.Sprintf("problem with syncFrom operation (%s/%s -> %s)",
			b.name, prefix, local)
		grip.Alert(m)
		return errors.Wrap(err, m)
	}

	grip.Infof("completed pull operation from %s/%s -> %s", b.name, prefix, local)
	return nil
}
