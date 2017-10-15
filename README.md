# Description

This is the golang version of Mendel's Accountant, a genetic mutation tracking program used to simulate and study macroevolution.

# Setup

- Install go, set GOROOT and GOPATH
- Install glide
- Install make
- Clone this repo into $GOPATH/src/github.com/genetic-algorithms/mendel-go

# Build and run

Build and run with test/input/mendel-case.ini:

```
make
```

Run with a different input file:

```
./mendel-go -f <input-file>
```

Build and run the automated tests:

```
make test-main
```

Test some of the packages:

```
make test-pkgs
```

# View godoc info of the project

Assuming you have cloned this project into `$GOPATH/src/github.com/genetic-algorithms/mendel-go`:

```
godoc github.com/genetic-algorithms/mendel-go
```

Then add any of the packages/subdirectories listed to the cmd above.
