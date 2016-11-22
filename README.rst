=================================================
``curator`` -- Artifact and Repository Management
=================================================

Overview
--------

Curator is a tool that we use at MongoDB to generate our package
repositories (e.g. ``repo.mongodb.org`` and
``repo.mongodb.com``). Additionally, curator provides tooling to
support the automated publication and management of build artifacts as
part of our continuous integration system.

Components
----------

Please refer to the ``--help`` output for ``curator`` and its
sub-commands. The following sections provides overviews of these
components and their use.

S3 Tools
~~~~~~~~

Curator includes a basic set of AWS S3 operations on the
command line. Although the get, put, ankd delete operations are faily
basic the ``sync-to`` and ``sync-from`` operations provide parallel
directory tree sync operations between s3 buckets and the local file
system. These operations compare objects using MD5 checksums, do
multi-threaded file uploads, and retry failed operations with
exponential backoff, for efficient and robust transfers.

Repobuilder
~~~~~~~~~~~

The repobuilder is an amalgamated operation that builds RPM and DEB
package repositories in S3. These jobs: sync files from an existing
repository, add packages from the local filesystem to the repository,
sign packages (for RPM), regenerate package metadata, sign package
metadata, generate html pages for web-based display, and sync the
changed files to the remote repository.

The current implementation of the repobuilder process depends on
external repository generation tools (e.g. ``createrepo`` and
``apt`` tools.) Additionally, the repobuilder currently depends on
MongoDB's internal signing service.

Index Pages
~~~~~~~~~~~

The index page rebuilding tool generates Apache-style directory
listing pages for a tree of files. This operation is part of the
repobuilder, but is available via an independent interface. This makes
it possible to regenerate listings for directories that are not
regenerated as part of the normal repository building process.

Artifacts
~~~~~~~~~

The artifacts functionality uses the release metadata feeds
(e.g. ``https://downloads.mongodb.org/full.json``) to fetch and
extract release build artifacts for local use. It is particularly
useful for fetching artifacts for and maintaining local caches of
MongoDB builds for multiple releases. Set the
``CURATOR_ARTIFACTS_DIRECTORY`` environment variable or pass the
``--path`` option to a flag, and then use the ``curator artifacts
download`` command to download files.

The ``artifacts`` command also includes two exploration subcommands
for discovering available builds: Use the ``list-map`` for specific
lists of available edition, target, and architecture combinations and
``list-all`` for a list of available target and architectures. Both
list operations are specific to a single version.

Combine the artifacts tool with the prune tool to avoid unbounded
cache growth.

Prune
~~~~~

Prune is based on the `lru <https://github.com/tychoish/lru>`_
library, and takes a file system tree and removes files, based on age,
until the total size of the files is less than a specified maximum
size. Prune uses modified time for age, in an attempt to have
consistent behavior indepenent of operation system and file system.

There are two modes of operation, a recursive mode which removes
objects from the tree recursively, but skips directory objects, and
directory mode, which does not collect objects recursively, but tracks
the size for the contents--recursively--of top-level directories.

Development
-----------

Design Principles and Goals
~~~~~~~~~~~~~~~~~~~~~~~~~~~

- To the greatest extent possible, maintain support for go 1.4
  (e.g. gccgo 5.3) as well as the latest stable release of mainline go
  (gc) toolchains. The Curator makefile uses a vendoring strategy that
  is naively compatible with both approaches.

- All operations in the continuous integration environment should be
  easily reproducible outside of the environment. In practice, curator
  exists to build repositories inside of Evergreen tasks; however, it
  is possible to run all stages of the repository building process by
  hand. The manual abilities are useful and required for publishing
  package revisions and repairing corrupt repositories.

- Leverage, as possible, third party libraries and tools. For example,
  the cache pruning and artifact management functionality is entirely
  derived from third-party repositories maintained separately from
  curator.

- Major functionality is implemented and executed in terms of `amboy
  <https://github.com/mongodb/amboy>`_ jobs and queues. Not only does
  this provide a framework for task execution, but leaves the door
  open to provide curator functionality as a highly available service
  with minimal modification.

APIs and Documentation
~~~~~~~~~~~~~~~~~~~~~~

See the `godoc API documentation <http://godoc.org/github.com/mongodb/curator>`_
for more information about curator interfaces and internals.

Issues
~~~~~~

Please file all issues in the `MAKE project
<https://jira.mongodb.org/browse/MAKE>`_ in the `MongoDB Jira
<https://jira.mongodb.org/>`_ instance.
