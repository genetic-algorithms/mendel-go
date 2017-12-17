# Description

This is the golang version of Mendel's Accountant, a genetic mutation tracking program used to simulate and study macroevolution.
It models genetic change over time by tracking each mutation that enters the simulated population from generation to generation
to the end of the simulation.
The software models each individual in the population, including their chromosomes, linkage blocks, and deleterious, favorable, and neutral mutations.
It supports several different models for mutation rate, mutation fitness distribution, selection, chromosome crossover, and population growth that are used in the
genetics field. The Mendel simulation also supports input parameters for many realistic genetic factors, including: reproduction rate, percentage of favorable,
deleterious and near-neutral mutations, fitness magnitude of mutations, selection noise, genome size, population size, and number of generations.
Mendel has been tuned to be able to simulate billions of mutations over 1000's of generations.

# Setup

- Install git, make
- Install go, set GOROOT and GOPATH:
```bash
GO_LATEST_STABLE=1.9.2   # find at https://golang.org/dl/
cd /usr/local
sudo wget https://storage.googleapis.com/golang/go${GO_LATEST_STABLE}.linux-amd64.tar.gz
sudo tar -xzf go${GO_LATEST_STABLE}.linux-amd64.tar.gz
# set in ~/.bash_profile: export PATH=$PATH:/usr/local/go/bin; export GOROOT=/usr/local/go; export GOPATH=$HOME/go
mkdir -p $GOPATH/src
```
- Install glide:
```bash
mkdir -p $HOME/bin $GOPATH/glide
cd $GOPATH/glide
GLIDE_LATEST_STABLE=0.13.1   # find at https://github.com/Masterminds/glide/releases
wget https://github.com/Masterminds/glide/releases/download/v${GLIDE_LATEST_STABLE}/glide-v${GLIDE_LATEST_STABLE}-linux-amd64.tar.gz
tar -xzvf glide-v${GLIDE_LATEST_STABLE}-linux-amd64.tar.gz
ln -s $GOPATH/glide/linux-amd64/glide $HOME/bin
```
- Clone mendel-go:
```bash
mkdir $GOPATH/src/github.com/genetic-algorithms
cd $GOPATH/src/github.com/genetic-algorithms
git clone git@github.com:genetic-algorithms/mendel-go.git
cd mendel-go
make mendel-go

```

# Build and Run

Build and run with test/input/mendel-case1.ini:

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
