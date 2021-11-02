===================================
``greenbay`` -- System Testing Tool
===================================

Overview
--------

We use ``greenbay`` tests to verify that build images in `Evergreen
<https://evergreen.mongodb.com/>`_, our Continuous Integration system, are
configured as we expect. The ``greenbay`` tool (`API documentation
<https://godoc.org/github.com/mongodb/greenbay>`_) reads a configuration
file and executes the tests it describes.

Writing Tests
-------------

A test must include a name, a list of suites to run, a test type, and some
args. For example, this verifies that pip is installed: ::

  tests:
    - name: pip_installed
      suites:
        - archlinux
      type: shell-operation
      args:
        command: "pip --version"

Save this as ``test.yaml``, and run it by referring to either the test or the
suite.
::

  greenbay run --conf test.yaml --test pip_installed
  greenbay run --conf test.yaml --suite archlinux 

You can pass ``--test`` or ``--suite`` multiple times. The ``--suite``
argument is useful for bundling tests together. If you don't pass either
``--test`` or ``--suite``, ``greenbay`` will run the "all" suite.

Common Test Types
-----------------

Every command must succeed:

::

  - name: command_test
    suites:
      - all
    type: command-group-all
    args:
      commands:
        - command: "run this command"
        - command: "and also this command"
        - command: "all should exit 0"

Check if a package is installed with dpkg:

::

  - name: dpkg_test
    suites:
      - all
    type: dpkg-installed
    args:
      package: package-name

Check if a file exists:

::

  - name: file_test
    suites:
      - all
    type: file-exists
    args:
      name: "/path/to/file"

Run a bash script:

::

  - name: bash_test
    suites:
      - all
    type: run-bash-script
    args:
      source: |
        # some bash to run
      output: "foo"

Run a python script:

::

  - name: python_test
    suites:
      - all
    type: run-program-system-python
    args:
      source: |
        print("howdy")
      output: "howdy"

Run a single command:

::

  - name: pip_test
    suites:
      - all
    type: shell-operation
    args:
      command: "pip --version"

At least one of the yum packages must be installed:

::

  - name: yum_group_test
    suites:
      - all
    type: yum-group-any
    args:
      packages:
        - glibc-devel.i386
        - glibc-devel.i686

Check if a package is installed with yum:

::

  - name: yum_test
    suites:
      - all
    type: yum-installed
    args:
      package: package-name

Greenbay Test Types
-------------------

You can list all ``greenbay`` test types with the following command: ::

  greenbay list

This will output a list of tests like this one: ::

  address-size
  brew-group-all
  brew-group-any
  brew-group-none
  brew-group-one
  brew-installed
  brew-not-installed
  command-group-all
  command-group-any
  command-group-none
  command-group-one
  compile-and-run-gcc-auto
  compile-and-run-gcc-system
  compile-and-run-go-auto
  compile-and-run-opt-go-default
  compile-and-run-toolchain-gccgo-v2
  compile-and-run-toolchain-v0
  compile-and-run-toolchain-v1
  compile-and-run-toolchain-v2
  compile-and-run-user-local-go
  compile-and-run-usr-local-go
  compile-and-run-visual-studio
  compile-gcc-auto
  compile-gcc-system
  compile-go-auto
  compile-opt-go-default
  compile-toolchain-gccgo-v2
  compile-toolchain-v0
  compile-toolchain-v1
  compile-toolchain-v2
  compile-user-local-go
  compile-usr-local-go
  compile-visual-studio
  dpkg-group-all
  dpkg-group-any
  dpkg-group-none
  dpkg-group-one
  dpkg-installed
  dpkg-not-installed
  file-does-not-exist
  file-exists
  file-group-all
  file-group-any
  file-group-none
  file-group-one
  gem-group-all
  gem-group-any
  gem-group-none
  gem-group-one
  gem-installed
  gem-not-installed
  irp-stack-size
  lxc-containers-configured
  open-files
  pacman-group-all
  pacman-group-any
  pacman-group-none
  pacman-group-one
  pacman-installed
  pacman-not-installed
  pip-group-all
  pip-group-any
  pip-group-none
  pip-group-one
  pip-installed
  pip-not-installed
  python-module-version
  run-bash-script
  run-bash-script-succeeds
  run-dash-script
  run-dash-script-succeeds
  run-program-gcc-auto
  run-program-gcc-system
  run-program-go-auto
  run-program-opt-go-default
  run-program-python-auto
  run-program-system-python
  run-program-system-python2
  run-program-system-python3
  run-program-toolchain-gccgo-v2
  run-program-toolchain-v0
  run-program-toolchain-v1
  run-program-toolchain-v2
  run-program-user-local-go
  run-program-usr-bin-pypy
  run-program-usr-local-go
  run-program-usr-local-python
  run-program-visual-studio
  run-sh-script
  run-sh-script-succeeds
  run-zsh-script
  run-zsh-script-succeeds
  shell-operation
  shell-operation-error
  yum-group-all
  yum-group-any
  yum-group-none
  yum-group-one
  yum-installed
  yum-not-installed
