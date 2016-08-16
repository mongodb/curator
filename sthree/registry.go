package sthree

import (
	"os"
	"runtime"
	"sync"

	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	"github.com/tychoish/grip"
)

// buckets is the internal and global cache.
var buckets *bucketRegistry

// bucketRegistry provides a cache of bucket
// references. bucketRegistry and its methods are all
// internal/private. There are public functions that access the global
// "buckets" instance, but the methods "SetCredentials" and
// "GetBucket" provide public wrappers around/for the global registry.
type bucketRegistry struct {
	m map[string]*Bucket
	l *sync.Mutex
	c AWSConnectionConfiguration
}

func init() {
	buckets = newBucketRegistry()
	buckets.init()
}

// Constructors and initialization methods for bucket registry.

func newBucketRegistry() *bucketRegistry {
	return &bucketRegistry{
		l: &sync.Mutex{},
		m: make(map[string]*Bucket),
		c: AWSConnectionConfiguration{},
	}
}

func (r *bucketRegistry) init() {
	auth, err := aws.EnvAuth()
	grip.CatchDebug(err)
	if err != nil {
		auth, err = aws.SharedAuth()
		grip.CatchWarning(err)
	}

	r.l.Lock()
	defer r.l.Unlock()

	r.c.Auth = auth
	r.c.Region = aws.USEast
}

// SetCredentials allows users to change the default credentials that
// new Bucket instances have upon creation. By default these objects
// read the AWS_ACCESS_KEY_ID and AWS_ACCESS_KEY environment variables
// and then fall back to reading from the "$HOME/.aws/credentials"
// file (using the "default" profile unless the AWS_PROFILE
// environment variable is set.)
func SetCredentials(c AWSConnectionConfiguration) {
	buckets.setCredentials(c)
}

func (r *bucketRegistry) setCredentials(c AWSConnectionConfiguration) {
	r.l.Lock()
	defer r.l.Unlock()

	if c.Region.Name == "" {
		c.Region = r.c.Region
	}

	r.c = c
}

// GetBucket takes the name of a bucket and returns a Bucket
// object. Creates a new Bucket object if one does not exist using the
// default credentials (see SetCredentials) for more information.
func GetBucket(name string) *Bucket {
	return buckets.getBucket(name)
}

func (r *bucketRegistry) getBucket(name string) *Bucket {
	r.l.Lock()
	defer r.l.Unlock()

	return r.getBucketWithCredentials(name, r.c)
}

// GetBucketWithProfile makes it possible to get a Bucket instance
// that uses credentials from a non-default AWS profile.
func GetBucketWithProfile(name, profile string) *Bucket {
	return buckets.getBucketFromProfile(name, profile)
}

func (r *bucketRegistry) getBucketFromProfile(name, account string) *Bucket {
	r.l.Lock()
	defer r.l.Unlock()

	existing := os.Getenv("AWS_PROFILE")
	if existing == "" {
		return r.getBucketWithCredentials(name, r.c)
	}

	grip.CatchError(os.Setenv("AWS_PROFILE", account))
	auth, err := aws.SharedAuth()
	grip.CatchWarning(err)
	creds := AWSConnectionConfiguration{
		Auth:   auth,
		Region: r.c.Region,
	}
	grip.CatchError(os.Setenv("AWS_PROFILE", existing))

	return r.getBucketWithCredentials(name, creds)
}

////////////////////////////////////////////////
//
// Internal interface used by the Bucket constructor/destructor
// methods to add and remove buckets from the registry/pool.
//
////////////////////////////////////////////////

func (r *bucketRegistry) registerBucket(b *Bucket) {
	r.l.Lock()
	defer r.l.Unlock()

	if _, ok := r.m[b.name]; ok {
		grip.Warningf("registering bucket named '%s', which already exists", b.name)
	}

	r.m[b.name] = b
}

func (r *bucketRegistry) getBucketWithCredentials(name string, creds AWSConnectionConfiguration) *Bucket {
	b, ok := r.m[name]
	if !ok {
		client := s3.New(r.c.Auth, r.c.Region)
		b = &Bucket{
			NewFilePermission: s3.BucketOwnerFull,
			s3:                client,
			bucket:            client.Bucket(name),
			credentials:       creds,
			name:              name,
			registry:          r,
			numJobs:           runtime.NumCPU() * 4,
			numRetries:        20,
		}
		grip.Noticef("creating new connection to bucket '%s'", name)

		r.m[name] = b
	}

	return b
}

func (r *bucketRegistry) removeBucket(b *Bucket) {
	r.l.Lock()
	defer r.l.Unlock()

	delete(r.m, b.name)
}
