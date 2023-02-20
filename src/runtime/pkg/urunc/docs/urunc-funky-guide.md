# Prerequisite
## Install [kata-container](https://github.com/kata-containers/kata-containers/blob/main/docs/Developer-Guide.md)
- version	
  ```
	/usr/local/bin/containerd-shim-kata-v2  --version
		Kata Containers containerd shim: id: "io.containerd.kata.v2"
		version: 3.1.0-alpha0
		commit:8b04ba95cb590fd11f1fbddfebf3f27b8a585c91
	/usr/local/bin/kata-runtime --version
		kata-runtime  : 3.1.0-alpha0
		commit   : 8b04ba95cb590fd11f1fbddfebf3f27b8a585c91
		OCI specs: 1.0.2-dev
   ```

### Build and install the Kata Containers runtime

```bash
$ git clone https://github.com/kata-containers/kata-containers.git
$ pushd kata-containers/src/runtime
$ make && sudo -E "PATH=$PATH" make install
$ sudo mkdir -p /etc/kata-containers/
$ sudo install -o root -g root -m 0640 /usr/share/defaults/kata-containers/configuration.toml /etc/kata-containers
$ popd
```

The build will create the following:

- runtime binary: `/usr/local/bin/kata-runtime` and `/usr/local/bin/containerd-shim-kata-v2`
- configuration file: `/usr/share/defaults/kata-containers/configuration.toml` and `/etc/kata-containers/configuration.toml`

### Configure to use rootfs image  
Make sure you have uncommented `image = /usr/share/kata-containers/kata-containers.img`(By default uncommented)
in your configuration file, commenting out the `initrd` line. For example:

```bash
$ sudo sed -i 's/^\(initrd =.*\)/# \1/g' /etc/kata-containers/configuration.toml
```
The rootfs image is created as shown below.

#### Build a rootfs image

```bash
$ pushd  kata-containers/tools/osbuilder/image-builder
$ script -fec 'sudo -E USE_DOCKER=true ./image_builder.sh "${ROOTFS_DIR}"'
$ popd
```

> **Notes:**
>
> - You must ensure that the *default Docker runtime* is `runc` to make use of
>   the `USE_DOCKER` variable. If that is not the case, remove the variable
>   from the previous command. See [Checking Docker default runtime](#checking-docker-default-runtime).
> - If you do *not* wish to build under Docker, remove the `USE_DOCKER`
>   variable in the previous command and ensure the `qemu-img` command is
>   available on your system.
>   - If `qemu-img` is not installed, you will likely see errors such as `ERROR: File /dev/loop19p1 is not a block device` and `losetup: /tmp/tmp.bHz11oY851: Warning: file is smaller than 512 bytes; the loop device may be useless or invisible for system tools`. These can be mitigated by installing the `qemu-img` command (available in the `qemu-img` package on Fedora or the `qemu-utils` package on Debian).
> - If `loop` module is not probed, you will likely see errors such as `losetup: cannot find an unused loop device`. Execute `modprobe loop` could resolve it.


#### Install the rootfs image

```bash
$ pushd kata-containers/tools/osbuilder/image-builder
$ commit="$(git log --format=%h -1 HEAD)"
$ date="$(date +%Y-%m-%d-%T.%N%z)"
$ image="kata-containers-${date}-${commit}"
$ sudo install -o root -g root -m 0640 -D kata-containers.img "/usr/share/kata-containers/${image}"
$ (cd /usr/share/kata-containers && sudo ln -sf "$image" kata-containers.img)
$ popd
```


### Enable seccomp

Enable seccomp as follows:

```bash
$ sudo sed -i '/^disable_guest_seccomp/ s/true/false/' /etc/kata-containers/configuration.toml
```

This will pass container seccomp profiles to the kata agent.


### Enable SELinux on the guest

> **Note:**
>
> - To enable SELinux on the guest, SELinux MUST be also enabled on the host.
> - You MUST create and build a rootfs image for SELinux in advance.
>   See [Create a rootfs image](#create-a-rootfs-image) and [Build a rootfs image](#build-a-rootfs-image).
> - SELinux on the guest is supported in only a rootfs image currently, so
>   you cannot enable SELinux with the agent init (`AGENT_INIT=yes`) yet.

Enable guest SELinux in Enforcing mode as follows:

```
$ sudo sed -i '/^disable_guest_selinux/ s/true/false/g' /etc/kata-containers/configuration.toml
```

The runtime automatically will set `selinux=1` to the kernel parameters and `xattr` option to
`virtiofsd` when `disable_guest_selinux` is set to `false`.

If you want to enable SELinux in Permissive mode, add `enforcing=0` to the kernel parameters.


  
### Install guest kernel images

#### Build Kata Containers Kernel

This document explains the steps to build a kernel recommended for use with
Kata Containers. To do this use `build-kernel.sh`, this script
automates the process to build a kernel for Kata Containers.

#### Requirements

The Linux kernel scripts further require a few packages (flex, bison, and libelf-dev)
See the CI scripts for your distro for more information...

#### Usage

```
$ ./build-kernel.sh -h
Overview:

	Build a kernel for Kata Containers

Usage:

	build-kernel.sh [options] <command> <argument>

Commands:

- setup

- build

- install

Options:

	-a <arch>       : Arch target to build the kernel, such as aarch64/ppc64le/s390x/x86_64.
	-c <path>   	: Path to config file to build the kernel.
	-d          	: Enable bash debug.
	-e          	: Enable experimental kernel.
	-f          	: Enable force generate config when setup.
	-g <vendor> 	: GPU vendor, intel or nvidia.
	-h          	: Display this help.
	-k <path>   	: Path to kernel to build.
	-p <path>   	: Path to a directory with patches to apply to kernel, only patches in top-level directory are applied.
	-t <hypervisor>	: Hypervisor_target.
	-v <version>	: Kernel version to use if kernel path not provided.
```

Example:
```bash
$ ./build-kernel.sh -v 5.10.25 -g nvidia -f -d setup
```
> **Note**
> - `-v 5.10.25`: Specify the guest kernel version. 
> - `-g nvidia`: To build a guest kernel supporting Nvidia GPU.
> - `-f`: The `.config` file is forced to be generated even if the kernel directory already exists.
> - `-d`: Enable bash debug mode.

> **Hint**: When in doubt look at [versions.yaml](../../../versions.yaml) to see what kernel version CI is using.


#### Setup kernel source code

```bash
$ git clone github.com/kata-containers/kata-containers
$ cd kata-containers/tools/packaging/kernel
$ ./build-kernel.sh setup
```

The script `./build-kernel.sh` tries to apply the patches from
`${GOPATH}/src/github.com/kata-containers/kata-containers/tools/packaging/kernel/patches/` when it
sets up a kernel. If you want to add a source modification, add a patch on this
directory. Patches present in the top-level directory are applied, with subdirectories being ignored.

The script also adds a kernel config file from
`${GOPATH}/src/github.com/kata-containers/kata-containers/tools/packaging/kernel/configs/` to `.config`
in the kernel source code. You can modify it as needed.

#### Build the kernel

After the kernel source code is ready, it is possible to build the kernel.

```bash
$ ./build-kernel.sh build
```

#### Install the Kernel in the default path for Kata

Kata Containers uses some default path to search a kernel to boot. To install
on this path, the following command will install it to the default Kata
containers path (`/usr/share/kata-containers/`).

```bash
$ sudo ./build-kernel.sh install
```

#### Submit Kernel Changes

Kata Containers packaging repository holds the kernel configs and patches. The
config and patches can work for many versions, but we only test the
kernel version defined in the [Kata Containers versions file][kata-containers-versions-file].

For further details, see [the kernel configuration documentation](configs).




### Install a hypervisor

When setting up Kata using a [packaged installation method](install/README.md#installing-on-a-linux-system), the
`QEMU` VMM is installed automatically. Cloud-Hypervisor and Firecracker VMMs are available from the [release tarballs](https://github.com/kata-containers/kata-containers/releases), as well as through [`kata-deploy`](../tools/packaging/kata-deploy/README.md).
You may choose to manually build your VMM/hypervisor.

#### Build a custom QEMU

Kata Containers makes use of upstream QEMU branch. 
Find the correct version of QEMU from the versions file:
```bash
$ source kata-containers/tools/packaging/scripts/lib.sh
$ qemu_version="$(get_from_kata_deps "assets.hypervisor.qemu.version")"
$ echo "${qemu_version}"
v6.2.0
```
Get source from the matching branch of QEMU:
```bash
$ git clone -b "${qemu_version}" https://github.com/qemu/qemu.git
$ your_qemu_directory="$(realpath qemu)"
```

There are scripts to manage the build and packaging of QEMU. For the examples below, set your
environment as:
```bash
$ packaging_dir="$(realpath kata-containers/tools/packaging)"
```

Kata often utilizes patches for not-yet-upstream and/or backported fixes for components,
including QEMU. These can be found in the [packaging/QEMU directory](../tools/packaging/qemu/patches),
and it's *recommended* that you apply them. For example, suppose that you are going to build QEMU
version 6.2.0, do:
```bash
$ "$packaging_dir/scripts/apply_patches.sh" "$packaging_dir/qemu/patches/6.2.x/"
```

To build utilizing the same options as Kata, you should make use of the `configure-hypervisor.sh` script. For example:
```bash
$ pushd "$your_qemu_directory"
$ "$packaging_dir/scripts/configure-hypervisor.sh" kata-qemu > kata.cfg
$ eval ./configure "$(cat kata.cfg)"
$ make -j $(nproc --ignore=1)
# Optional
$ sudo -E make install
$ popd
```

If you do not want to install the respective QEMU version, the configuration file can be modified to point to the correct binary. In `/etc/kata-containers/configuration.toml`, change `path = "/path/to/qemu/build/qemu-system-x86_64"` to point to the correct QEMU binary.





### Build `virtiofsd`

When using the file system type virtio-fs (default), `virtiofsd` is required

```bash
$ pushd kata-containers/tools/packaging/static-build/virtiofsd
$ ./build.sh
$ popd
```

Modify `/etc/kata-containers/configuration.toml` and update value `virtio_fs_daemon = "/path/to/kata-containers/tools/packaging/static-build/virtiofsd/virtiofsd/virtiofsd"` to point to the binary.



### Check hardware requirements

You can check if your system is capable of creating a Kata Container by running the following:

```bash
$ sudo kata-runtime check
```

If your system is *not* able to run Kata Containers, the previous command will error out and explain why.












## Install [urunc](https://github.com/nubificus/kata-containers/blob/feat_kata_urunc/src/runtime/pkg/urunc/docs/kata-urunc-dev-guide.md)

- version
  ```
	/usr/local/bin/containerd-shim-kata-unikernels-v2 --version
	Kata Containers containerd shim: id: "io.containerd.kata.v2"
	version: 2.5.0-alpha0
	commit:c371d6aed21a91fe2e941899c92f794a7593bbf1-dirty
  ```


### Instructions / Commands for kata-urunc dev

#### Dev environment setup

This assumes you already have a working kata-qemu installation (following the blog posts' instructions). If this is not the case, you may need to edit some directories accordingly.

First we need to get the source code and build the `containerd-shim` binary:

```bash
export PATH=$PATH:$(go env GOPATH)/bin && \
  export GOPATH=$(go env GOPATH) && \
  export GO111MODULE=off

go get -d -u github.com/nubificus/kata-containers
pushd $GOPATH/src/github.com/nubificus/kata-containers/src/runtime
git switch feat_kata_urunc # or git switch -c feat_kata_urunc
export GO111MODULE=on
export PREFIX=/opt/unikata
make
sudo -E PATH=$PATH -E PREFIX=$PREFIX make install
sudo mv /opt/unikata/bin/containerd-shim-kata-v2 /opt/unikata/containerd-shim-kata-unikernels-v2
sudo rm -rf /opt/unikata/bin && sudo rm -rf /opt/unikata/share
sudo ln -s /opt/unikata/containerd-shim-kata-unikernels-v2 /usr/local/bin
popd
```

Next we need to create a valid config file. For now, this is done by copying the QEMU config file and adding an extra `unikernel` option:

```bash
sudo cp /opt/kata/configs/configuration.toml /opt/kata/configs/configuration-urunc.toml
```

Update kata's configuration file located in `/opt/kata/configs/configuration-urunc.toml`

```
[hypervisor.urunc]
unikernel=true
path = "/usr/bin/qemu-system-x86_64"
kernel = "/usr/share/kata-containers/vmlinux.container"
image = "/usr/share/kata-containers/kata-containers.img"
machine_type = "q35"
```
you can find an example config file in [configuration-urunc.toml](../config/kata/configuration-urunc.toml)

Create a new file somewhere in PATH eg: `/usr/local/bin/containerd-shim-kata-urunc-v2` and add the followning lines:

```bash
#!/bin/bash
KATA_CONF_FILE=/opt/kata/configs/configuration-urunc.toml /usr/local/bin/containerd-shim-kata-unikernels-v2 $@
```

Then make it executable.

Last but not least, add the following lines to the containerd config file in `/etc/containerd/config.toml`:

```
[plugins.cri.containerd.runtimes]
  [plugins.cri.containerd.runtimes.kata-urunc]
    runtime_type = "io.containerd.kata-urunc.v2"
```

you can find an example config file in [config.toml](../config/containerd/config.toml)

Restart `containerd`:

```bash
sudo systemctl restart containerd
```

Finally, Copy the funky-monitor `ukvm` in `~/includos/x86_64/lib/ukvm-bin` to `/opt/kata/bin/`
```
cp ~/funkyos/includos/x86_64/lib/ukvm-bin /opt/kata/bin/solo5-hvt
```

#### Recompile:

After any changes, you can recompile the binary using the following commands. It is helpful if you place them in a simple script:

```bash
export PATH=$PATH:$(go env GOPATH)/bin && \
  export GOPATH=$(go env GOPATH)

pushd $GOPATH/src/github.com/nubificus/kata-containers/src/runtime
git switch unikernels-support # or git switch -c unikernels-support
export GO111MODULE=on
export PREFIX=/opt/unikata
make
sudo -E PATH=$PATH -E PREFIX=$PREFIX make install
sudo mv /opt/unikata/bin/containerd-shim-kata-v2 /opt/unikata/containerd-shim-kata-unikernels-v2
sudo rm -rf /opt/unikata/bin && sudo rm -rf /opt/unikata/share
popd
```

#### Run test containers:

To run a test container, we need to get the image and run it using the `kata-urunc` runtime:

```bash
sudo ctr images pull docker.io/library/ubuntu:latest
sudo ctr run --runtime io.containerd.run.kata-urunc.v2 -t --rm docker.io/library/ubuntu:latest ubuntu-kata-test uname -a
sudo ctr run --runtime io.containerd.run.kata.v2 -t --rm docker.io/library/ubuntu:latest ubuntu-kata-test uname -a

sudo ctr run --runtime io.containerd.run.kata-urunc.v2 -t --rm docker.io/urunc/testhello:latest urunc-kata-test /unikernel/hello
```


##### Clean up

```bash
sudo pkill -f ubuntu-kata-test ;\
  sudo ctr c rm ubuntu-kata-test ;\
  sudo ctr snapshot rm ubuntu-kata-test ;\
  sudo systemctl restart containerd ;\
  sudo rm -rf /run/containerd/io.containerd.runtime.v2.task/default/ubuntu-kata-test
```

```bash
sudo pkill -f urunc-kata-test ;\
  sudo ctr c rm urunc-kata-test ;\
  sudo ctr snapshot rm urunc-kata-test ;\
  sudo systemctl restart containerd ;\
  sudo rm -rf /run/containerd/io.containerd.runtime.v2.task/default/urunc-kata-test
```

#### Clean up dead containers

During this whole process many things fail, so it often is required to clean up the dead containers:

```bash
sudo ctr snapshots rm ubuntu-kata-test
sudo ctr c rm ubuntu-kata-test
# sometimes we need to kill the processes before removing the container
ps ax | grep ubuntu-kata-test
sudo kill ... ... ...
```

























## build [funky-app binary](https://github.com/AtsushiKoshiba/funky-unikernel/blob/vfpga/doc-funky/build-opencl-apps.md)

- version
  ```
	funky-v0.3.1-1-gb9286401c-dirty (x86_64 / 64-bit)
	commit: b9286401c63525de889d90fd5bc30b79a68709ca
  ```

The build.sh script located in `funky-unikernel/example/hello_xilinx` can be used to build most of the funky-app

  
If you see this message when you boot the app:
  ```
 +--> WARNiNG: Environment unsafe for production
 +--> Stop option enabled. Shutting down now...
  ```

Change the `for_production` option in the 297th line of `~/funkyos/includeos/post.service.cmake ` to `OFF`:

```
option(for_production "Stop the OS when conditions not suitable for production" OFF)
```


# Configuration
## Configure to use devmapper
To make it work you need to **prepare thin-pool** in advance and **update containerd's configuration file**. This file is typically located at `/etc/containerd/config.toml`.


```bash
  [plugins."io.containerd.snapshotter.v1.devmapper"]
    pool_name = "devpool"
    root_path = "/var/lib/containerd/devmapper"
    base_image_size = "10GB"
    discard_blocks = true
    async_remove = false
    fs_type = "ext2"
```

The following configuration flags are supported:

- `root_path` - a directory where the metadata will be available (if empty default location for containerd plugins will be used)
- `pool_name` - a name to use for the devicemapper thin pool. Pool name should be the same as in /dev/mapper/ directory
- `base_image_size` - defines how much space to allocate when creating the base device
- `async_remove` - flag to async remove device using snapshot GC's cleanup callback
- `discard_blocks` - whether to discard blocks when removing a device. This is especially useful for returning disk space to the filesystem when using loopback devices.
- `fs_type` - defines the file system to use for snapshot device mount. Valid values are ext4 and xfs. Defaults to ext4 if unspecified.
- `fs_options` - optionally defines the file system options. This is currently only applicable to ext4 file system.

### How to setup device mapper thin-pool
See [here](https://pkg.go.dev/github.com/containerd/containerd/snapshots/devmapper#section-readme)


# Build image
## Custom image creation for urunc

Use the image-builder script in `kata-containers/src/runtime/pkg/urunc/image-builder/build.sh` to build a container image 


```
# To see the help message:
./build.sh -h

# Create a urunc/testhello image containing a hello 
# and import it to ctr
mkdir data # stores data needed to run the app
./build.sh -u hello.binary -i urunc/testhello -e data -c

# Create a urunc/funky:hvt image containing a simple_add.hvt
# import it to ctr and run it using solo5
./build.sh -u simple_add.hvt -i urunc/funky:hvt -e data -c

# Create a urunc/funky:qemu image containing a simple_add.qemu
# import it to ctr and run it using solo5
./build.sh -u simple_add.qemu -i urunc/funky:qemu -e data -c

```

urunc supports tree different binarys:

- `binary`: normal gcc compiled binary
- `hvt`: compiled targeting solo5-hvt
- `qemu`: compiled targeting qemu

Depends on the suffix of the binary, the binary will be run in different way.

move the bitstream file (e.g: `krnl_vadd.xclbin`) into `data` directory so that urunc know the input argument 

# Run

```
sudo ctr run --snapshotter devmapper --runtime io.containerd.kata-urunc.v2 --rm docker.io/urunc/funky:hvt FunkyosTest  /unikernel/simple_add.hvt 
```







# Troubleshoot Kata Containers

1. `./../vendor/golang.org/x/sys/unix/sysvshm_unix.go:33:7: unsafe.Slice requires go1.17 or later (-lang was set to go1.16; check go.mod)`

Solution: change go version in go.mod, then run `go mod tidy, go mod vendor`

2. `ctr: failed to create shim: Could not bind mount /run/kata-containers/shared/sandboxes/test-kata/mounts to   /run/kata-containers/shared/sandboxes/test-kata/shared: no such file or directory: unknown`

Reason: umount failed
Solution: 
```
# For example the app name is FunkyosTest
sudo rm -rf /run/containerd/io.containerd.runtime.v2.task/default/FunkyosTest
sudo umount /run/kata-containers/shared/sandboxes/FunkyosTest/shared

# Add the above command in stopContainer Function solves this problem
```

3. `ctr: failed to create shim: no such file or directory: not found`

	run the command again resolved this error.
