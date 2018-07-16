Summary: Mendel's Accountant genetic evolution simulation
Name: mendel-go
Version: %{getenv:VERSION}
Release: 1
Epoch: 1
License: GNU GPL v3
Source: mendel-go-%{version}.tar.gz
Packager: Bruce Potter
#Vendor: ?
#Distribution: ?
Prefix: /usr
#BuildRoot: ?
BuildArch: x86_64

%description
Mendel's Accountant performs biologically realistic genetic evolution simulation.

%prep
%setup -q

%build
# This phase is done in ~/rpmbuild/BUILD/mendel-go-1.0.0. All of the tarball source has been unpacked there and
# is in the same file structure as it is in the git repo. $RPM_BUILD_DIR has a value like ~/rpmbuild/BUILD
#env | grep -i build
# Need to play some games to get our src dir under a GOPATH
rm ../src; ln -s . ../src
mkdir -p ../github.com/genetic-algorithms
rm ../github.com/genetic-algorithms/mendel-go; ln -s ../../mendel-go-1.0.0 ../github.com/genetic-algorithms/mendel-go

GOPATH=$RPM_BUILD_DIR make mendel-go

%install
# The install phase puts all of the files in the paths they should be in when the rpm is installed on a system.
# The $RPM_BUILD_ROOT is a simulated root file system and usually has a value like: ~/rpmbuild/BUILDROOT/mendel-go-1.0.0-1.x86_64
#TODO: put this in the Makefile instead
rm -rf $RPM_BUILD_ROOT
mkdir -p $RPM_BUILD_ROOT%{prefix}/bin $RPM_BUILD_ROOT%{prefix}/share/mendel-go
cp mendel-go $RPM_BUILD_ROOT%{prefix}/bin
cp mendel-defaults.ini LICENSE COPYRIGHT $RPM_BUILD_ROOT%{prefix}/share/mendel-go

%files
#%defattr(-, root, root)
#%doc LICENSE COPYRIGHT
/usr/bin/mendel-go
/usr/share/mendel-go

%clean
# This step happens *after* the %files packaging
rm -rf $RPM_BUILD_ROOT
