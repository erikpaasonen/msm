package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/msmhq/msm/internal/config"
	"github.com/msmhq/msm/internal/logging"
	"github.com/msmhq/msm/internal/world"
)

func (s *Server) IsRunning() bool {
	return s.Screen.IsRunning()
}

func (s *Server) Start() error {
	if s.IsRunning() {
		return fmt.Errorf("server %q is already running", s.Name)
	}

	jarPath := s.JarPath()
	if _, err := os.Stat(jarPath); os.IsNotExist(err) {
		return fmt.Errorf("jar file not found: %s", jarPath)
	}

	if err := s.SetupRAMWorlds(); err != nil {
		return fmt.Errorf("failed to set up RAM worlds: %w", err)
	}

	invocation := s.Config.Invocation
	invocation = strings.ReplaceAll(invocation, "{RAM}", strconv.Itoa(s.Config.RAM))
	invocation = strings.ReplaceAll(invocation, "{JAR}", s.Config.JarPath)

	if err := s.Screen.Start(s.Path, invocation, s.Config.Username); err != nil {
		return err
	}

	if hasRAM, _ := AnyRAMWorldsConfigured(s.GlobalCfg); hasRAM {
		if err := StartSyncDaemon(s.GlobalCfg); err != nil {
			logging.Warn("failed to start sync daemon", "error", err)
		}
	}

	return nil
}

func (s *Server) SetupRAMWorlds() error {
	if !s.GlobalCfg.RamdiskStorageEnabled {
		return nil
	}

	worlds, err := world.DiscoverAll(
		s.Path,
		s.Name,
		s.GlobalCfg,
		s.Config.WorldStoragePath,
		s.Config.WorldStorageInactivePath,
	)
	if err != nil {
		return err
	}

	for _, w := range worlds {
		if w.InRAM && w.Active {
			logging.Debug("setting up RAM symlink", "world", w.Name, "server", s.Name)
			if err := w.SetupRAMSymlink(s.Config.WorldStoragePath); err != nil {
				return fmt.Errorf("failed to set up RAM symlink for world %q: %w", w.Name, err)
			}
		}
	}

	return nil
}

func (s *Server) Stop(immediate bool) error {
	if !s.IsRunning() {
		return nil
	}

	if !immediate {
		delay := s.Config.StopDelay
		msg := strings.ReplaceAll(s.Config.MessageStop, "{DELAY}", strconv.Itoa(delay))

		s.Say(msg)
		time.Sleep(time.Duration(delay) * time.Second)
	}

	if err := s.SendCommand("stop"); err != nil {
		return err
	}

	for i := 0; i < 30; i++ {
		if !s.IsRunning() {
			s.maybeStopSyncDaemon()
			return nil
		}
		time.Sleep(1 * time.Second)
	}

	if err := s.Screen.Kill(); err != nil {
		return err
	}

	s.maybeStopSyncDaemon()
	return nil
}

func (s *Server) maybeStopSyncDaemon() {
	if !IsSyncDaemonRunning() {
		return
	}

	running, err := AnyServersRunning(s.GlobalCfg)
	if err != nil {
		logging.Warn("failed to check running servers", "error", err)
		return
	}

	if !running {
		if err := StopSyncDaemon(); err != nil {
			logging.Warn("failed to stop sync daemon", "error", err)
		}
	}
}

func (s *Server) Restart(immediate bool) error {
	if s.IsRunning() {
		if !immediate {
			delay := s.Config.RestartDelay
			msg := strings.ReplaceAll(s.Config.MessageRestart, "{DELAY}", strconv.Itoa(delay))

			s.Say(msg)
			time.Sleep(time.Duration(delay) * time.Second)
		}

		if err := s.Stop(true); err != nil {
			return err
		}

		time.Sleep(2 * time.Second)
	}

	return s.Start()
}

func (s *Server) Status() string {
	if s.IsRunning() {
		return "running"
	}
	return "stopped"
}

func (s *Server) Console() error {
	if !s.IsRunning() {
		return fmt.Errorf("server %q is not running", s.Name)
	}

	return s.Screen.AttachAsUser(s.Config.Username)
}

func (s *Server) SendCommand(cmd string) error {
	return s.Screen.SendCommandAsUser(cmd, s.Config.Username)
}

func (s *Server) Say(message string) error {
	return s.SendCommand(fmt.Sprintf("say %s", message))
}

func (s *Server) Kick(player, reason string) error {
	if reason != "" {
		return s.SendCommand(fmt.Sprintf("kick %s %s", player, reason))
	}
	return s.SendCommand(fmt.Sprintf("kick %s", player))
}

func (s *Server) SaveAll() error {
	return s.SendCommand("save-all")
}

func (s *Server) SaveOff() error {
	return s.SendCommand("save-off")
}

func (s *Server) SaveOn() error {
	return s.SendCommand("save-on")
}

func Create(name string, cfg *config.Config) (*Server, error) {
	serverPath := filepath.Join(cfg.ServerStoragePath, name)

	if _, err := os.Stat(serverPath); !os.IsNotExist(err) {
		return nil, fmt.Errorf("server %q already exists", name)
	}

	if err := os.MkdirAll(serverPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create server directory: %w", err)
	}

	worldStoragePath := filepath.Join(serverPath, cfg.DefaultWorldStoragePath)
	if err := os.MkdirAll(worldStoragePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create world storage directory: %w", err)
	}

	worldStorageInactivePath := filepath.Join(serverPath, cfg.DefaultWorldStorageInactivePath)
	if err := os.MkdirAll(worldStorageInactivePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create inactive world storage directory: %w", err)
	}

	return Load(serverPath, name, cfg)
}

func Delete(name string, cfg *config.Config) error {
	serverPath := filepath.Join(cfg.ServerStoragePath, name)

	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		return fmt.Errorf("server %q not found", name)
	}

	server, err := Load(serverPath, name, cfg)
	if err != nil {
		return err
	}

	if server.IsRunning() {
		return fmt.Errorf("server %q is running; stop it first", name)
	}

	return os.RemoveAll(serverPath)
}

func Rename(oldName, newName string, cfg *config.Config) error {
	oldPath := filepath.Join(cfg.ServerStoragePath, oldName)
	newPath := filepath.Join(cfg.ServerStoragePath, newName)

	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return fmt.Errorf("server %q not found", oldName)
	}

	if _, err := os.Stat(newPath); !os.IsNotExist(err) {
		return fmt.Errorf("server %q already exists", newName)
	}

	server, err := Load(oldPath, oldName, cfg)
	if err != nil {
		return err
	}

	if server.IsRunning() {
		return fmt.Errorf("server %q is running; stop it first", oldName)
	}

	return os.Rename(oldPath, newPath)
}
