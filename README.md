# Description

This is the golang version of [Mendel](https://en.wikipedia.org/wiki/Gregor_Mendel)'s Accountant, a genetic mutation tracking program used to simulate and study macroevolution in a biologically realistic way.
It models genetic change over time by tracking each mutation that enters the simulated population from generation to generation
to the end of the simulation.
The software models each individual in the population, including their chromosomes, linkage blocks, and deleterious, favorable, and neutral mutations.
It supports several different models for mutation rate, mutation fitness distribution, selection, chromosome crossover, and population growth that are used in the
genetics field. The Mendel simulation also supports input parameters for many realistic genetic factors, including: reproduction rate, percentage of favorable,
deleterious and near-neutral mutations, fitness magnitude of mutations, selection noise, genome size, population size, and number of generations.
Mendel has been optimized to be able to simulate billions of mutations over 1000's of generations.

# Running mendel-go

If you want to run mendel-go (w/o building it from source) see the [Mendel wiki page](https://github.com/genetic-algorithms/mendel-go/wiki).

# Build mendel-go From Source

- Install: git, make
- [Install go](https://golang.org/doc/install)
- Add `go` to your `PATH` and set `GOPATH` environment variable in `~/.bash_profile` or `~/.profile`:

  ```bash
  export PATH=$PATH:/usr/local/go/bin
  export GOPATH=$HOME
  ```

- Install glide:

  ```bash
  curl https://glide.sh/get | sh
  ```
 
- Clone and build mendel-go:

  ```bash
  mkdir -p $GOPATH/src/github.com/genetic-algorithms
  cd $GOPATH/src/github.com/genetic-algorithms
  git clone git@github.com:genetic-algorithms/mendel-go.git
  cd mendel-go
  make mendel-go
  ```

# Run mendel-go

Run (building if necessary) with test/input/case1.ini:

```
make
```

Run with a different input file:

```
./mendel-go -f <input-file>
```

Build and run the automated tests:

```
make test
```

Test some of the packages:

```
make test-pkgs
```

# Build the mendel-go Packages

To publish a new version of mendel-go, first build the RPM and macOS packages of it:

```bash
make rpmbuild
make macpkg
```

Create a new release at https://github.com/genetic-algorithms/mendel-go/releases and upload the packages you just built.

# Build and Run the Mendel Web UI

See the [mendel-web-ui git repo](https://github.com/genetic-algorithms/mendel-web-ui) to build and run the Mendel web UI.

# View godoc info of the project

Assuming you have cloned this project into `$GOPATH/src/github.com/genetic-algorithms/mendel-go`:

```
godoc github.com/genetic-algorithms/mendel-go
```

Then add any of the packages/subdirectories listed to the cmd above.
