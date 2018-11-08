=================
Curator Artifacts
=================

Overview
--------

The ``artifacts`` sub-command of `curator <https://github.com/mongodb/curator>`_
provides an interface for downloading and managing large numbers of MongoDB
versions, typically within evergreen.

Begin by getting a version of ``curator`` for your system from the `curator
evergreen project <https://evergreen.mongodb.com/waterfall/curator>`_.

Before using curator you'll want to set the ``CURATOR_ARTIFACTS_DIRECTORY``
environment variable. On Evergreen hosts, use ``/data/curator-cache``, but a
location within ``/tmp`` often makes sense. Internally, ``curator`` use the
`download info feed <http://downloads.mongodb.org/full.json>`_, which it will
cache for 4 hours.

Version Discovery
-----------------

MongoDB versions have three properties: a processor architecture or "arch", an
edition that "enterprise", "targeted" (includes SSL, and is tarted at a specific
operating system), and "base" (generic static builds.) ``curator`` will attempt to
find the right version, but it tends to be somewhat conservative and opts for
base versions in more cases than you might expect. Use the ``list-map`` command,
as in the following example to discover the available builds for a given
version: ::

    ./curator artifacts list-map --version 3.2.17

The response format is, as follows: ::

    3.2.17
	 target='linux_x86_64', edition='base', arch='x86_64'
	 target='osx', edition='base', arch='x86_64'
	 target='linux_i686', edition='base', arch='i686'
	 target='windows_x86_64', edition='base', arch='x86_64'
	 target='windows_i686', edition='base', arch='i386'
	 target='windows_x86_64-2008plus', edition='base', arch='x86_64'
	 target='amzn64', edition='enterprise', arch='x86_64'
	 target='rhel62', edition='enterprise', arch='x86_64'
	 target='suse11', edition='enterprise', arch='x86_64'
	 target='ubuntu1404', edition='enterprise', arch='x86_64'
	 target='ubuntu1204', edition='enterprise', arch='x86_64'
	 target='windows', edition='enterprise', arch='x86_64'
	 target='rhel57', edition='enterprise', arch='x86_64'
	 target='osx-ssl', edition='base', arch='x86_64'
	 target='windows_x86_64-2008plus-ssl', edition='base', arch='x86_64'
	 target='rhel70', edition='enterprise', arch='x86_64'
	 target='debian71', edition='enterprise', arch='x86_64'
	 target='ubuntu1404', edition='targeted', arch='x86_64'
	 target='ubuntu1204', edition='targeted', arch='x86_64'
	 target='rhel70', edition='targeted', arch='x86_64'
	 target='rhel62', edition='targeted', arch='x86_64'
	 target='suse11', edition='targeted', arch='x86_64'
	 target='debian71', edition='targeted', arch='x86_64'
	 target='amazon', edition='targeted', arch='x86_64'
	 target='rhel55', edition='targeted', arch='x86_64'
	 target='osx', edition='enterprise', arch='x86_64'
	 target='suse12', edition='enterprise', arch='x86_64'
	 target='suse12', edition='targeted', arch='x86_64'
	 target='debian81', edition='enterprise', arch='x86_64'
	 target='debian81', edition='targeted', arch='x86_64'
	 target='rhel71', edition='enterprise', arch='ppc64le'
	 target='ubuntu1604', edition='enterprise', arch='x86_64'
	 target='ubuntu1604', edition='targeted', arch='x86_64'

You can get the same information in a JSON format, using the following command: ::

    ./curator artifacts list-variants --version 3.2.17

The response has the following format: ::

    {
	  "version": "3.2.17",
	  "targets": [
	     "linux_x86_64",
	     "osx",
	     "linux_i686",
	     "windows_x86_64",
	     "windows_i686",
	     "windows_x86_64-2008plus",
	     "amzn64",
	     "rhel62",
	     "suse11",
	     "ubuntu1404",
	     "ubuntu1204",
	     "windows",
	     "rhel57",
	     "osx-ssl",
	     "windows_x86_64-2008plus-ssl",
	     "rhel70",
	     "debian71",
	     "amazon",
	     "rhel55",
	     "suse12",
	     "debian81",
	     "rhel71",
	     "ubuntu1604"
	  ],
	  "editions": [
	     "base",
	     "enterprise",
	     "targeted"
	  ],
	  "architectures": [
	     "x86_64",
	     "i686",
	     "i386",
	     "ppc64le"
	  ]
       }

In some older versions of MongoDB the "enterprise" build is refered to as the
"subscriber" version. Additionally, in recent versions, the set of targeted
builds has been substantially similar to the enterprise builds, but that is less
true in older versions. Between these two commands you can discover the
information you need to download versions.

Downloading
-----------

Official Releases
~~~~~~~~~~~~~~~~~

Use the download command to download a version into the cache: ::

    ./curator artifacts download --version 3.2.17

The output for this operation is as follows: ::

    [curator] 2018/01/19 13:05:04 [p=info]: job server running
    [curator] 2018/01/19 13:05:04 [p=info]: waiting for 1 download jobs to complete
    [curator] 2018/01/19 13:05:04 [p=notice]: downloading: /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-3.2.17.tgz
    [curator] 2018/01/19 13:05:05 [p=notice]: downloaded /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-3.2.17.tgz file
    [curator] 2018/01/19 13:05:09 [p=notice]: extracted archive: /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-3.2.17.tgz
    [curator] 2018/01/19 13:05:09 [p=info]: all download tasks complete, processing errors now

You can repeat this operation multiple times, and ``curator`` will only download
the artifact once, evident from the output of a repeated operation: ::

    [curator] 2018/01/19 13:04:56 [p=info]: job server running
    [curator] 2018/01/19 13:04:56 [p=info]: waiting for 1 download jobs to complete
    [curator] 2018/01/19 13:04:56 [p=notice]: file /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-3.2.17.tgz is already downloaded
    [curator] 2018/01/19 13:04:56 [p=info]: all download tasks complete, processing errors now

``curator`` will download multiple packages in parallel, to the greatest extent
possible: ::

    ./curator artifacts download --version 3.2.17 --version 3.7.1 --version 1.8.4

Consider the following output: ::

    [curator] 2018/01/19 13:09:49 [p=info]: job server running
    [curator] 2018/01/19 13:09:49 [p=info]: waiting for 3 download jobs to complete
    [curator] 2018/01/19 13:09:49 [p=notice]: file /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-3.2.17.tgz is already downloaded
    [curator] 2018/01/19 13:09:49 [p=notice]: downloading: /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-1.8.4.tgz
    [curator] 2018/01/19 13:09:49 [p=notice]: downloading: /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-3.7.1.tgz
    [curator] 2018/01/19 13:09:52 [p=notice]: downloaded /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-1.8.4.tgz file
    [curator] 2018/01/19 13:09:52 [p=notice]: downloaded /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-3.7.1.tgz file
    [curator] 2018/01/19 13:09:53 [p=notice]: extracted archive: /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-1.8.4.tgz
    [curator] 2018/01/19 13:09:56 [p=notice]: extracted archive: /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-3.7.1.tgz
    [curator] 2018/01/19 13:09:56 [p=info]: all download tasks complete, processing errors now

There's no way to download multiple targets/architectures at once, but you can
try, with the following operation: ::

    ./curator artifacts download --version 3.2.17 --version 3.7.1 --target rhel62 --edition targeted

Consider the following output: ::

    [curator] 2018/01/19 13:12:04 [p=info]: job server running
    [curator] 2018/01/19 13:12:04 [p=info]: waiting for 2 download jobs to complete
    [curator] 2018/01/19 13:12:04 [p=notice]: downloading: /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-rhel62-3.7.1.tgz
    [curator] 2018/01/19 13:12:04 [p=notice]: downloading: /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-rhel62-3.2.17.tgz
    [curator] 2018/01/19 13:12:07 [p=notice]: downloaded /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-rhel62-3.7.1.tgz file
    [curator] 2018/01/19 13:12:12 [p=notice]: extracted archive: /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-rhel62-3.7.1.tgz
    [curator] 2018/01/19 13:12:15 [p=notice]: downloaded /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-rhel62-3.2.17.tgz file
    [curator] 2018/01/19 13:12:19 [p=notice]: extracted archive: /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-rhel62-3.2.17.tgz
    [curator] 2018/01/19 13:12:19 [p=info]: all download tasks complete, processing errors now

Special Versions
~~~~~~~~~~~~~~~~

Curator allows two "special" version string forms to allow you to access
specific versions of MongoDB. To access the latest successful build of a version
(e.g. the "nightly") for a branch, use a version argument such as one of the
following: ::

    ./curator artifacts download --version 3.2-latest --version 3.4-latest

Consider the following output: ::

    [curator] 2018/01/19 13:17:10 [p=info]: job server running
    [curator] 2018/01/19 13:17:10 [p=info]: waiting for 2 download jobs to complete
    [curator] 2018/01/19 13:17:10 [p=notice]: downloading: /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-v3.2-latest.tgz
    [curator] 2018/01/19 13:17:10 [p=notice]: downloading: /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-v3.4-latest.tgz
    [curator] 2018/01/19 13:17:12 [p=notice]: downloaded /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-v3.2-latest.tgz file
    [curator] 2018/01/19 13:17:13 [p=notice]: downloaded /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-v3.4-latest.tgz file
    [curator] 2018/01/19 13:17:16 [p=notice]: extracted archive: /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-v3.2-latest.tgz
    [curator] 2018/01/19 13:17:17 [p=notice]: extracted archive: /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-v3.4-latest.tgz
    [curator] 2018/01/19 13:17:17 [p=info]: all download tasks complete, processing errors now

These are always development releases and always reflect builds from commits to
the branches, for variants that have passed.

The ``latest`` feature does not work for development (i.e. odd release series)
which are always built from master.

The ``current`` (this is also alised to ``stable``) is useful for return the
latest official build for a release series, as in the following example: ::

    ./curator artifacts download --version 3.7-current --version 3.2-current --version 3.4-current

Consider the following output: ::

    [curator] 2018/01/19 14:04:21 [p=info]: job server running
    [curator] 2018/01/19 14:04:21 [p=info]: waiting for 3 download jobs to complete
    [curator] 2018/01/19 14:04:21 [p=notice]: downloading: /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-3.4.10.tgz
    [curator] 2018/01/19 14:04:21 [p=notice]: file /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-3.7.1.tgz is already downloaded
    [curator] 2018/01/19 14:04:21 [p=notice]: downloading: /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-3.2.18.tgz
    [curator] 2018/01/19 14:04:23 [p=notice]: downloaded /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-3.4.10.tgz file
    [curator] 2018/01/19 14:04:25 [p=notice]: downloaded /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-3.2.18.tgz file
    [curator] 2018/01/19 14:04:27 [p=notice]: extracted archive: /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-3.4.10.tgz
    [curator] 2018/01/19 14:04:29 [p=notice]: extracted archive: /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-3.2.18.tgz
    [curator] 2018/01/19 14:04:29 [p=info]: all download tasks complete, processing errors now

Local Discovery
---------------

Once you have downloaded the versions you want to access the ``list-all`` and
``get-path`` commands may be useful for taking advantage of your cache. The
``list-all`` command returns an account of your current cache: ::

    ./curator artifacts list-all

This returns a data structure that maps paths in your artifact cache that hold
extracted MongoDB builds to metadata about that build: ::

    {
       "/home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-1.8.4": {
	  "version": "1.8.4",
	  "options": {
	     "target": "linux",
	     "arch": "x86_64",
	     "edition": "base",
	     "debug": false
	  }
       },
       "/home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-3.2.17": {
	  "version": "3.2.17",
	  "options": {
	     "target": "linux",
	     "arch": "x86_64",
	     "edition": "base",
	     "debug": false
	  }
       },
       "/home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-3.7.1": {
	  "version": "3.7.1",
	  "options": {
	     "target": "linux",
	     "arch": "x86_64",
	     "edition": "base",
	     "debug": false
	  }
       },
       "/home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-rhel62-3.2.17": {
	  "version": "3.2.17",
	  "options": {
	     "target": "rhel62",
	     "arch": "x86_64",
	     "edition": "targeted",
	     "debug": false
	  }
       },
       "/home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-rhel62-3.7.1": {
	  "version": "3.7.1",
	  "options": {
	     "target": "rhel62",
	     "arch": "x86_64",
	     "edition": "targeted",
	     "debug": false
	  }
       },
       "/home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-v3.2-latest": {
	  "version": "3.2-latest",
	  "options": {
	     "target": "linux",
	     "arch": "x86_64",
	     "edition": "base",
	     "debug": false
	  }
       },
       "/home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-v3.4-latest": {
	  "version": "3.4-latest",
	  "options": {
	     "target": "linux",
	     "arch": "x86_64",
	     "edition": "base",
	     "debug": false
	  }
       }
    }

You can also use the ``get-path`` to get access to these paths, using the same
arguments that you'd pass to the ``download`` path, for use in shell scripting,
as in: ::

    ./curator artifacts get-path --version 3.2.17

The result of this command is: ::

    /home/evgdev/mdb/curator/artifacts/curator-artifact-cache/mongodb-linux-x86_64-3.2.17

Cache Pruning
-------------

Presumably you don't want to maintain your own local cache of builds forever,
and curator has a cache pruning tool available for your use, with the ``prune``
sub-command. ``prune`` is a top-level sub-command, _not_ a sub-command of
``artifacts``. The ``prune`` operation uses and respects the
``CURATOR_ARTIFACTS_DIRECTORY`` environment variable.

These examples assume that ``CURATOR_ARTIFACTS_DIRECTORY`` is set.

``prune`` supports an LRU cache model. Having said that, it tracks *modified
time* (mtime) of the file system objects, which means you should generally
use the ``touch`` command to touch the enclosing directories when you use them,
as in the following operation: ::

    ./curator prune --max-size 1000

The ``--max-size`` operation specifies a maximum size, and prune will delete
the oldest directories until the total size of the cached files are. It ignores
the ``full.json`` file, but does *not* skip any other files.

If you specify the ``--recursive`` option it will look at *all* file objects
recursively (cleaning up empty directories as needed,) but not removing a
directory. For artifact caches, you almost never want this.
