package world

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/msmhq/msm/pkg/screen"
)

func (w *World) ToRAM(user string) error {
	if !w.GlobalCfg.RamdiskStorageEnabled {
		return fmt.Errorf("ramdisk storage is not enabled")
	}

	if err := os.MkdirAll(w.RAMPath, 0755); err != nil {
		return fmt.Errorf("failed to create RAM directory: %w", err)
	}

	flagPath := w.FlagPath()
	rsyncCmd := fmt.Sprintf("rsync -rt --exclude '%s' '%s/' '%s'",
		filepath.Base(flagPath), w.Path, w.RAMPath)

	return screen.RunAsUser(user, rsyncCmd)
}

func (w *World) ToDisk(user string) error {
	if !w.InRAM {
		return nil
	}

	flagPath := w.FlagPath()
	rsyncCmd := fmt.Sprintf("rsync -rt --exclude '%s' '%s/' '%s'",
		filepath.Base(flagPath), w.RAMPath, w.Path)

	return screen.RunAsUser(user, rsyncCmd)
}

func (w *World) ToggleRAM(user string) error {
	if !w.GlobalCfg.RamdiskStorageEnabled {
		return fmt.Errorf("ramdisk storage is not enabled")
	}

	flagPath := w.FlagPath()

	if w.InRAM {
		fmt.Printf("Synchronising world %q to disk... ", w.Name)
		if err := w.ToDisk(user); err != nil {
			return err
		}
		fmt.Println("Done.")

		fmt.Printf("Removing RAM flag from world %q... ", w.Name)
		if err := os.Remove(flagPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		fmt.Println("Done.")

		fmt.Printf("Removing world %q from RAM... ", w.Name)
		if err := os.RemoveAll(w.RAMPath); err != nil {
			return err
		}
		fmt.Println("Done.")

		w.InRAM = false
	} else {
		fmt.Printf("Adding RAM flag to world %q... ", w.Name)
		if err := touchFile(flagPath); err != nil {
			return err
		}
		fmt.Println("Done.")

		fmt.Printf("Copying world %s to RAM... ", w.Name)
		w.RAMPath = filepath.Join(w.GlobalCfg.RamdiskStoragePath, w.ServerName, w.Name)
		if err := w.ToRAM(user); err != nil {
			return err
		}
		fmt.Println("Done.")

		w.InRAM = true
	}

	fmt.Println("Changes will only take effect after server is restarted.")
	return nil
}

func (w *World) SetupRAMSymlink(worldStoragePath string) error {
	if !w.InRAM || !w.GlobalCfg.RamdiskStorageEnabled {
		return nil
	}

	serverWorldPath := worldStoragePath
	if !filepath.IsAbs(worldStoragePath) {
		serverWorldPath = filepath.Join(w.ServerPath, worldStoragePath)
	}

	symlinkPath := filepath.Join(serverWorldPath, w.Name)

	linkTarget, err := os.Readlink(symlinkPath)
	if err == nil && linkTarget == w.RAMPath {
		return nil
	}

	if _, err := os.Lstat(symlinkPath); err == nil {
		if err := os.RemoveAll(symlinkPath); err != nil {
			return fmt.Errorf("failed to remove existing world directory: %w", err)
		}
	}

	if err := os.Symlink(w.RAMPath, symlinkPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

func touchFile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	return file.Close()
}

func rsyncAvailable() bool {
	_, err := exec.LookPath("rsync")
	return err == nil
}
