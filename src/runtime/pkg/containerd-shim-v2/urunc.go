package containerdshim

import (
	"context"
	"encoding/json"
	"path/filepath"
	"io"
	"io/ioutil"
	"net"
	osexec "os/exec"
	"strings"
	"time"
	"github.com/containerd/containerd/api/types/task"
	"github.com/kata-containers/kata-containers/src/runtime/virtcontainers"
	"github.com/sirupsen/logrus"
)

// {"cmdline":"redis-server","net":{"if":"ukvmif0","cloner":"True","type":"inet","method":"static","addr":"10.10.10.2","mask":"16"}}

type HvtArgsNetwork struct {
	If     string `json:"if"`
	Cloner string `json:"cloner"`
	Type   string `json:"type"`
	Method string `json:"method"`
	Addr   string `json:"addr"`
	Mask   string `json:"mask"`
	Gw     string `json:"gw"`
}

type HvtArgsBlock struct {
	Source string `json:"source"`
	Path   string `json:"path"`
	Fstype string `json:"fstype"`
	Mount  string `json:"mountpoint"`
}

type HvtArgs struct {
	Cmdline string         `json:"cmdline"`
	Net     HvtArgsNetwork `json:"net"`
	Blk     HvtArgsBlock   `json:"blk,omitempty"`
	Env     []string       `json:"env,omitempty"`
	Cwd     string         `json:"cwd,omitempty"`
	Mem     string         `json:"mem,omitempty"`
}

type Command struct {
	cmdString string
	container *container
	id        string
	stdin     string
	stdout    string
	stderr    string
	bundle    string
	exec      *osexec.Cmd
}

func CmdLine(execData virtcontainers.ExecData) string {

	// BinaryType string
	// BinaryPath string
	// IPAddress  string
	// Mask       string
	// Tap        string
	// Container  *Container
	logF := logrus.Fields{"src": "uruncio", "file": "cs/urunc.go", "func": "CmdLine"}
	shimLog.WithField("BinaryType", execData.BinaryType).WithFields(logF).Error("ExecData")
	shimLog.WithField("BinaryPath", execData.BinaryPath).WithFields(logF).Error("ExecData")
	shimLog.WithField("IPAddress", execData.IPAddress).WithFields(logF).Error("ExecData")
	shimLog.WithField("Mask", execData.Mask).WithFields(logF).Error("ExecData")
	shimLog.WithField("Tap", execData.Tap).WithFields(logF).Error("ExecData")
	shimLog.WithField("Container", execData.Container.ID()).WithFields(logF).Error("ExecData")

	switch execData.BinaryType {
	case "pause":
		return execData.BinaryPath
	case "hvt":
		return HvtCmd(execData)
	case "qemu":
		return QemuCmd(execData)
	case "binary":
		return execData.BinaryPath
	default:
		return ""
	}
}

func HvtCmd(execData virtcontainers.ExecData) string {
	logF := logrus.Fields{"src": "uruncio", "file": "cs/urunc.go", "func": "HvtCmd"}
	logrus.WithFields(logF).Error("")
	ifaces, _ := net.Interfaces()
	execData.Tap = ifaces[len(ifaces)-1].Name
	execData.Tap = "tap100"
	nsParts := strings.Split(execData.NetNs, "/")
	ns := nsParts[len(nsParts)-1]
	logrus.WithFields(logF).WithField("NetNs1", execData.NetNs).Error("")
	logrus.WithFields(logF).WithField("NetNs2", ns).Error("")

	HvtMonitor := "/opt/kata/bin/solo5-hvt"
	// ./tenders/hvt/solo5-hvt --net:service0=tap192  tests/test_net/test_net.hvt
	// return HvtMonitor + "--net:service0=" + execData.Tap + " " + execData.BinaryPath
	// /solo5-hvt --net=tap100 -- redis.hvt '{"cmdline":"redis-server","net":{"if":"ukvmif0","cloner":"True","type":"inet","method":"static","addr":"10.10.10.2","mask":"16"}}'

	hvtNet := HvtArgsNetwork{
		If:     "ukvmif0",
		Cloner: "True",
		Type:   "inet",
		Method: "static",
		Addr:   execData.IPAddress,
		Mask:   "0",
		Gw:   execData.Gateway,
	}

	hvtBlock := HvtArgsBlock{
		Source:	"etfs",
		Path:	"/dev/ld0a",
		Fstype:	"blk",
		Mount:	"/data",
	}

	hvtArgs := HvtArgs{
		Cmdline: filepath.Base(execData.BinaryPath),//"redis-server",
		Net:     hvtNet,
		Blk:     hvtBlock,
	}

	b, _ := json.Marshal(hvtArgs)
	cmdString := ""
	nsString := ""
	if ns != "" {
		nsString = "ip netns exec " + ns + " "
	} else {
		nsString = ""
	}
	if execData.BlkDevice != "" {
		cmdString = nsString + HvtMonitor + " --net=" + execData.Tap + " --disk=" + execData.BlkDevice + " " + execData.BinaryPath + " " + "test"//string(b)
	} else {
		cmdString = nsString + HvtMonitor + " --net=" + execData.Tap + " " + execData.BinaryPath + " " + string(b)
	}

	// stripped := strings.Replace(cmdString, "\\", "", -1)
	// unquoted := fmt.Sprintln(stripped)
	// unqoted, err := strconv.Unquote(stripped)
	// if err != nil {
	// 	logrus.WithFields(logF).WithField("err", err.Error()).Error("")
	// }
	logrus.WithFields(logF).WithField("cmdline", string(b)).Error("")
	logrus.WithFields(logF).WithField("cmd", cmdString).Error("")
	// cmdParts := strings.Split(stripped, " ")

	// name, args := cmdParts[0], cmdParts[1:]
	// output, _ := osexec.Command(name, args...).CombinedOutput()
	// logrus.WithFields(logF).WithField("out", string(output)).Error("")

	logrus.WithFields(logF).WithField("ifaces", len(ifaces)).Error("")
	for _, iface := range ifaces {
		logrus.WithFields(logF).WithField("name", iface.Name).Error("")
	}

	// output, _ := osexec.Command(name, args...).CombinedOutput()
	// logrus.WithFields(logF).WithField("out", string(output)).Error("")\
	// na doume kai to GW mhpws xreiazetai!

	return cmdString

}

func QemuCmd(execData virtcontainers.ExecData) string {
	qemuCmd := "qemu-system-x86_64 -cpu host"
	qemuCmd += " -enable-kvm"
	qemuCmd += " -m 128"
	qemuCmd += " -nodefaults -no-acpi "
	qemuCmd += " -display none -serial stdio "
	qemuCmd += " -device isa-debug-exit "
	qemuCmd += " -net nic,model=virtio "
	qemuCmd += " -net tap,script=no,ifname=" + execData.Tap
	qemuCmd += " -kernel " + execData.BinaryPath
	qemuCmd += " -append \"netdev.ipv4_addr=" + execData.IPAddress + "netdev.ipv4_gw_addr=" + execData.Gateway + " netdev.ipv4_subnet_mask=255.255.255.255 --"

	// qemu-system-x86_64 \
	//     -cpu host \
	//     -enable-kvm \
	//     -m 128 \
	//     -nodefaults -no-acpi \
	//     -display none -serial stdio \
	//     -device isa-debug-exit \
	//     -net nic,model=virtio \
	//     -net tap,script=no,ifname=tap106 \
	//     -kernel /app-helloworld_kvm-x86_64 \
	//     -append "netdev.ipv4_addr=$IP netdev.ipv4_gw_addr=169.254.1.1 netdev.ipv4_subnet_mask=255.255.255.255 --"
	return qemuCmd

}

func CreateCommand(execData virtcontainers.ExecData, container *container) *Command {
	logF := logrus.Fields{"src": "uruncio", "file": "cs/urunc.go", "func": "CreateCommand"}
	cmdString := CmdLine(execData)
	files, _ := ioutil.ReadDir("/run/containerd/io.containerd.runtime.v2.task/default/"+ container.id +"/rootfs")
	for _, file := range files {
		if !file.IsDir(){
			if strings.HasSuffix(file.Name(),".xclbin"){
			    cmdString = cmdString+" "+file.Name()
			 } else {
			    continue
		    }
		}
	}
	shimLog.WithField("BinaryType", execData.BinaryType).WithFields(logF).Error("exec info")
	shimLog.WithField("cmdString", cmdString).WithFields(logF).Error("exec info")

	args := strings.Split(cmdString, " ")
	var newCmd *osexec.Cmd
	if len(args) == 1 {
		shimLog.WithField("Stringlen", len(args)).WithField("cmdString", args[0]).WithFields(logF).Error("exec info")
		newCmd = osexec.Command(args[0])
	} else {
		name, args := args[0], args[1:]
		newCmd = osexec.Command(name, args...)
		newCmd.Env = append(newCmd.Environ(), "XILINX_XRT=/opt/xilinx/xrt")
		newCmd.Env = append(newCmd.Environ(), "PATH=/opt/xilinx/xrt/bin:/opt/xilinx/xrt/bin:/opt/xilinx/xrt/bin:/opt/xilinx/xrt/bin:/opt/xilinx/xrt/bin:/opt/xilinx/xrt/bin:/opt/xilinx/xrt/bin:/home/zirong/.local/bin:/home/zirong/.pyenv/plugins/pyenv-virtualenv/shims:/home/zirong/.pyenv/shims:/home/zirong/.nix-profile/bin:/nix/var/nix/profiles/default/bin:/home/zirong/.local/bin:/home/zirong/.pyenv/plugins/pyenv-virtualenv/shims:/home/zirong/.pyenv/bin:/home/zirong/.nix-profile/bin:/nix/var/nix/profiles/default/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games:/snap/bin:/home/zirong/funkyos//bin:/home/zirong/funkyos//bin:/usr/local/go/bin:/home/zirong/go/bin")
		newCmd.Env = append(newCmd.Environ(), "LD_LIBRARY_PATH=/opt/xilinx/xrt/lib:/opt/xilinx/xrt/lib:/opt/xilinx/xrt/lib:/opt/xilinx/xrt/lib:/opt/xilinx/xrt/lib:/opt/xilinx/xrt/lib:/opt/xilinx/xrt/lib:")
		newCmd.Env = append(newCmd.Environ(), "PYTHONPATH=/opt/xilinx/xrt/python:/opt/xilinx/xrt/python:/opt/xilinx/xrt/python:/opt/xilinx/xrt/python:/opt/xilinx/xrt/python:/opt/xilinx/xrt/python:/opt/xilinx/xrt/python:")
	}
	return &Command{cmdString: cmdString, container: container, id: container.id, stdin: container.stdin, stdout: container.stdout, stderr: container.stderr, bundle: container.bundle, exec: newCmd}
}

func (c *Command) ioPipes() (io.WriteCloser, io.ReadCloser, io.ReadCloser, error) {
	stdin, err := c.exec.StdinPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	stdout, err := c.exec.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	stderr, err := c.exec.StderrPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	return stdin, stdout, stderr, nil
}

func (c *Command) SetIO(ctx context.Context) error {
	logF := logrus.Fields{"src": "uruncio", "file": "cs/urunc.go", "func": "SetIO"}
	shimLog.WithFields(logF).WithField("path", c.exec.Path).Error("stdout, stderr redirected")

	stdin, stdout, stderr, err := c.ioPipes()
	shimLog.WithFields(logF).Error("ioPipes retrieved")

	if err != nil {
		shimLog.WithFields(logF).WithField("err", err.Error()).Error("ioPipes retrieved")

		return err
	}

	c.container.stdinPipe = stdin
	shimLog.WithFields(logF).Error("container stdin redirected")

	if c.container.stdin != "" || c.container.stdout != "" || c.container.stderr != "" {
		tty, err := newTtyIO(ctx, c.stdin, c.stdout, c.stderr, c.container.terminal)
		if err != nil {
			return err
		}
		c.container.ttyio = tty
		shimLog.WithFields(logF).Error("container ttyio set")

		go ioCopy(shimLog.WithField("container", c.id), c.container.exitIOch, c.container.stdinCloser, tty, stdin, stdout, stderr)
	}
	return nil
}
func (c *Command) Start() error {
	logF := logrus.Fields{"src": "uruncio", "file": "cs/urunc.go", "func": "Start"}

	err := c.exec.Start()
	shimLog.WithFields(logF).WithField("path", c.exec.Path).Error("CMD STARTED")
	c.container.status = task.StatusRunning
	return err
}

func (c *Command) Wait() error {
	logF := logrus.Fields{"src": "uruncio", "file": "cs/urunc.go", "func": "Wait"}

	shimLog.WithFields(logF).Error("Wait Start")
	time.Sleep(500 * time.Millisecond)

	c.exec.Wait()
	shimLog.WithFields(logF).Error("exec returned")

	shimLog.WithFields(logF).Error("cmd completed")

	shimLog.WithFields(logF).Error("exitIOch close start")
	close(c.container.exitIOch)
	shimLog.WithFields(logF).Error("exitIOch closed")

	close(c.container.stdinCloser)
	shimLog.WithFields(logF).Error("stdinCloser closed")

	c.container.status = task.StatusStopped
	shimLog.WithFields(logF).Error("container.status: StatusStopped")
	return nil
}
