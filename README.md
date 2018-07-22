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

- Install git, make
- Install go, set GOROOT and GOPATH:
```bash
GO_LATEST_STABLE=1.10.3   # find at https://golang.org/dl/
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
- Clone and build mendel-go:

```bash
mkdir $GOPATH/src/github.com/genetic-algorithms
cd $GOPATH/src/github.com/genetic-algorithms
git clone git@github.com:genetic-algorithms/mendel-go.git
cd mendel-go
make mendel-go

```

# Run mendel-go

Run (building if necessary) with test/input/mendel-case1.ini:

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

# Run in SPC and View Plot Files

[SPC](https://github.com/whbrewer/spc) is an open source platform for running scientific programs and visualization the output. You can download it to run your own instance of it, and install the [mendel-go SPC plugin](https://github.com/genetic-algorithms/mendel-go-spc), or you can do some small runs of mendel_go on the [public instance of SPC](http://ec2-52-43-51-28.us-west-2.compute.amazonaws.com:8580/myapps).

# View godoc info of the project

Assuming you have cloned this project into `$GOPATH/src/github.com/genetic-algorithms/mendel-go`:

```
godoc github.com/genetic-algorithms/mendel-go
```

Then add any of the packages/subdirectories listed to the cmd above.
