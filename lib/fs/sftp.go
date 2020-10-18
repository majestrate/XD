package fs

import (
	"encoding/base64"
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strings"
	"github.com/majestrate/XD/lib/log"
	"github.com/majestrate/XD/lib/util"
)

type sftpFile struct {
	f *sftp.File
}

func (f *sftpFile) Write(data []byte) (int, error) {
	return f.f.Write(data)
}

func (f *sftpFile) Sync() error {
	return nil
}

func (f *sftpFile) Read(data []byte) (int, error) {
	return f.f.Read(data)
}

func (f *sftpFile) WriteAt(data []byte, at int64) (n int, err error) {
	_, err = f.f.Seek(at, 0)
	if err == nil {
		n, err = f.Write(data)
		if err == nil {
			_, err = f.f.Seek(0, 0)
		}
	}
	return
}

func (f *sftpFile) ReadAt(data []byte, at int64) (n int, err error) {
	_, err = f.f.Seek(at, 0)
	if err == nil {
		n, err = f.Read(data)
		if err == nil {
			_, err = f.f.Seek(0, 0)
		}
	}
	return
}

func (f *sftpFile) Close() error {
	return f.f.Close()
}

type sftpFS struct {
	username   string
	hostname   string
	keyfile    string
	remotekey  string
	port       int
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

func (fs *sftpFS) ensureSSH() (*ssh.Client, error) {
	if fs.sshClient == nil {
		log.Debugf("read key %s", fs.keyfile)
		data, err := ioutil.ReadFile(fs.keyfile)
		if err != nil {
			return nil, err
		}
		log.Debugf("sftp parse key file")
		ourKey, err := ssh.ParsePrivateKey(data)
		if err != nil {
			return nil, err
		}
		k, err := base64.StdEncoding.DecodeString(fs.remotekey)
		if err != nil {
			return nil, err
		}
		theirKey, err := ssh.ParsePublicKey(k)
		if err != nil {
			return nil, err
		}
		addr := net.JoinHostPort(fs.hostname, fmt.Sprintf("%d", fs.port))
		log.Debugf("sftp dial to %s", addr)
		fs.sshClient, err = ssh.Dial("tcp", addr, &ssh.ClientConfig{
			User: fs.username,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(ourKey),
			},
			HostKeyCallback: ssh.FixedHostKey(theirKey),
		})
		return fs.sshClient, err
	}
	return fs.sshClient, nil
}

func (fs *sftpFS) ensureSFTP() (*sftp.Client, error) {
	if fs.sftpClient == nil {
		sshClient, err := fs.ensureSSH()
		if err == nil {
			fs.sftpClient, err = sftp.NewClient(sshClient)
		}
		return fs.sftpClient, err
	}
	return fs.sftpClient, nil
}

func (fs *sftpFS) Open() error {
	_, err := fs.ensureSSH()
	if err == nil {
		_, err = fs.ensureSFTP()
	}
	return err
}

func (fs *sftpFS) Close() (err error) {
	if fs.sftpClient != nil {
		err = fs.sftpClient.Close()
		fs.sftpClient = nil
	}
	if err == nil {
		if fs.sshClient != nil {
			err = fs.sshClient.Close()
			fs.sshClient = nil
		}

	}
	return
}

func (fs *sftpFS) ensureConn(visit func(*sftp.Client) error) error {
	s, err := fs.ensureSFTP()
	if err == nil {
		err = visit(s)
	} else {
		err = fs.Close()
	}
	return err
}

func (fs *sftpFS) EnsureDir(fname string) (err error) {
	mkdirParents := func(client *sftp.Client, dir string) (err error) {
		var parents string

		if path.IsAbs(dir) {
			// Otherwise, an absolute path given below would be turned in to a relative one
			// by splitting on "/"
			parents = "/"
		}

		for _, name := range strings.Split(dir, "/") {
			if name == "" {
				// Paths with double-/ in them should just move along
				// this will also catch the case of the first character being a "/", i.e. an absolute path
				continue
			}
			parents = path.Join(parents, name)
			if fs.FileExists(parents) {
				continue
			}
			err = client.Mkdir(parents)
			if status, ok := err.(*sftp.StatusError); ok {
				if status.Code == 4 {
					var fi os.FileInfo
					fi, err = client.Stat(parents)
					if err == nil {
						if !fi.IsDir() {
							return fmt.Errorf("File exists: %s", parents)
						}
					}
				}
			}
			if err != nil {
				break
			}
		}
		return
	}
	return fs.ensureConn(func(c *sftp.Client) error {
		return mkdirParents(c, fname)
	})
}

func (fs *sftpFS) FileExists(fname string) bool {
	if fs.sftpClient == nil {
		return false
	}
	_, err := fs.sftpClient.Stat(fname)
	return err == nil
}

func (fs *sftpFS) OpenFileReadOnly(fname string) (f ReadFile, err error) {
	err = fs.ensureConn(func(c *sftp.Client) error {
		var e error
		var osf *sftp.File
		osf, e = c.Open(fname)
		if e == nil {
			f = &sftpFile{osf}
		}
		return e
	})
	return
}

func (fs *sftpFS) OpenFileWriteOnly(fname string) (f WriteFile, err error) {
	err = fs.ensureConn(func(c *sftp.Client) error {
		var e error
		var osf *sftp.File
		osf, e = c.Create(fname)
		if e == nil {
			f = &sftpFile{osf}
		}
		return e
	})
	return
}

func (fs *sftpFS) Glob(glob string) (matches []string, err error) {
	err = fs.ensureConn(func(c *sftp.Client) error {
		var e error
		matches, e = c.Glob(glob)
		return e
	})
	return
}

func (fs *sftpFS) EnsureFile(fname string, sz uint64) error {
	if fs.FileExists(fname) {
		return nil
	}
	return fs.ensureConn(func(c *sftp.Client) error {
		d, _ := sftp.Split(fname)
		var err error
		if d != "" {
			err = fs.EnsureDir(d)
		}
		if err == nil {
			var f WriteFile
			f, err = fs.OpenFileWriteOnly(fname)
			if err == nil {
				if sz > 0 {
					_, err = io.CopyN(f, util.Zero, int64(sz))
				}
			}
			f.Close()
		}
		return err
	})
}

func (fs *sftpFS) removeAllDir(root string, c *sftp.Client) error {
	dirs, err := c.ReadDir(root)
	if err != nil {
		return err
	}
	for idx := range dirs {
		if dirs[idx].IsDir() {
			err = fs.removeAllDir(dirs[idx].Name(), c)
		} else {
			err = c.Remove(fs.Join(root, dirs[idx].Name()))
		}
		if err != nil {
			return err
		}
	}
	return c.RemoveDirectory(root)
}

func (fs *sftpFS) Join(paths ...string) string {
	p := ""
	err := fs.ensureConn(func(c *sftp.Client) error {
		p = c.Join(paths...)
		return nil
	})
	if err != nil {
		panic(err.Error())
	}
	return p
}

func (fs *sftpFS) Move(oldpath, newpath string) (err error) {
	dir, _ := fs.Split(newpath)
	err = fs.EnsureDir(dir)
	if err == nil {
		err = fs.ensureConn(func(c *sftp.Client) error {
			return c.Rename(oldpath, newpath)
		})
	}
	return
}

func (fs *sftpFS) Split(path string) (base, file string) {
	base, file = sftp.Split(path)
	return
}

func (fs *sftpFS) Remove(fpath string) error {
	return fs.ensureConn(func(c *sftp.Client) error {
		return c.Remove(fpath)
	})
}

func (fs *sftpFS) Stat(fpath string) (fi os.FileInfo, err error) {
	err = fs.ensureConn(func(c *sftp.Client) error {
		fi, err = c.Stat(fpath)
		return err
	})
	return
}

func (fs *sftpFS) RemoveAll(fpath string) error {
	return fs.ensureConn(func(c *sftp.Client) error {
		st, err := c.Stat(fpath)
		if err != nil {
			return err
		}
		if st.IsDir() {
			return fs.removeAllDir(fpath, c)
		}
		return c.Remove(fpath)
	})
}

func SFTP(username, hostname, keyfile, remotekey string, port int) Driver {
	return &sftpFS{
		username:  username,
		hostname:  hostname,
		keyfile:   keyfile,
		remotekey: remotekey,
		port:      port,
	}
}
