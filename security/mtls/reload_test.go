package mtls

import (
	"crypto/tls"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReloader_TriggersOnChange(t *testing.T) {
	tc := createTempCerts(t)
	defer os.RemoveAll(tc.Dir)

	// Сначала создадим серверный конфиг и повесим апдейтер на замену cert/CA
	conf, _, err := TLSConfigServer(Config{
		CACertPath:     tc.CAPath,
		CertPath:       tc.ServerCert,
		KeyPath:        tc.ServerKey,
		ReloadInterval: 0,
	})
	if err != nil {
		t.Fatalf("TLSConfigServer: %v", err)
	}

	triggered := make(chan struct{}, 1)
	r := NewReloader(Config{
		CACertPath: tc.CAPath,
		CertPath:   tc.ServerCert,
		KeyPath:    tc.ServerKey,
	}, func(nb *bundle) {
		// применяем новые материалы в tls.Config
		conf.ClientCAs = nb.rootPool
		conf.Certificates = []tls.Certificate{nb.cert}
		select {
		case triggered <- struct{}{}:
		default:
		}
	})

	// Стартуем с быстрым тикером
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	r.Start(ticker)
	defer r.Stop()

	// Перезапишем один из файлов, чтобы изменилась mtime.
	// Лучше CA — это безопаснее и не требует пересборки ключа.
	newCA := filepath.Join(tc.Dir, "ca2.pem")
	if err := os.WriteFile(newCA, mustRead(t, tc.CAPath), 0o644); err != nil {
		t.Fatalf("write new CA: %v", err)
	}
	// Обновим исходный CA-файл (подменим содержимое тем же, но с добавл. переносом)
	if f, err := os.OpenFile(tc.CAPath, os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
		_, _ = f.WriteString("\n")
		_ = f.Close()
	} else {
		t.Fatalf("append ca: %v", err)
	}

	// Ждём триггер
	select {
	case <-triggered:
		// ок
	case <-time.After(1 * time.Second):
		t.Fatalf("reloader did not trigger on file change")
	}
}

// маленький helper
func mustRead(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	return b
}

func TestReloader_Stop_Idempotent(t *testing.T) {
	t.Parallel()

	r := NewReloader(Config{}, func(*bundle) {})
	r.Stop()
	r.Stop()
}

func TestNewReloaderWithLogger_NilApply_NoPanic(t *testing.T) {
	t.Parallel()

	r := NewReloaderWithLogger(Config{}, nil, nil)
	if r == nil {
		t.Fatalf("expected non-nil reloader")
	}

	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("unexpected panic: %v", rec)
		}
	}()

	r.apply(nil)
}

func TestReloader_Start_NilTicker_LogsError(t *testing.T) {
	t.Parallel()

	errCh := make(chan error, 1)
	r := NewReloaderWithLogger(Config{}, func(*bundle) {}, func(ev ReloadEvent) {
		errCh <- ev.Err
	})

	r.Start(nil)

	select {
	case err := <-errCh:
		if !errors.Is(err, errNilTicker) {
			t.Fatalf("expected errNilTicker, got %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected nil ticker error to be logged")
	}
}

func TestReloader_NilReceiver_NoPanic(t *testing.T) {
	t.Parallel()

	var r *Reloader
	r.Start(nil)
	r.Stop()
}
