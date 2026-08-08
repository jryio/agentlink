package adopt

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
)

const (
	transactionVersion = 1
	maxTransactionSize = 4 << 10
)

type transaction struct {
	Version     int    `json:"version"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Overwrite   bool   `json:"overwrite"`

	record            string
	stage             string
	sourceBackup      string
	destinationBackup string
}

func newTransaction(source, destination string, overwrite bool) transaction {
	sum := sha256.Sum256([]byte(source + "\x00" + destination))
	id := fmt.Sprintf("%x", sum[:8])
	return transaction{
		Version:     transactionVersion,
		Source:      source,
		Destination: destination,
		Overwrite:   overwrite,
		record:      path.Join(".agents", ".agentlink-adopt-"+id+".json"),
		stage: path.Join(
			path.Dir(destination),
			"."+path.Base(destination)+".agentlink-adopt-"+id+".stage",
		),
		sourceBackup: path.Join(
			path.Dir(source),
			"."+path.Base(source)+".agentlink-adopt-"+id+".source",
		),
		destinationBackup: path.Join(
			path.Dir(destination),
			"."+path.Base(destination)+".agentlink-adopt-"+id+".destination",
		),
	}
}

func beginTransaction(root *os.Root, transaction transaction) error {
	for _, artifact := range []string{
		transaction.record,
		transaction.stage,
		transaction.sourceBackup,
		transaction.destinationBackup,
	} {
		_, exists, err := lstat(root, artifact)
		if err != nil {
			return fmt.Errorf("inspect adoption workspace %s: %w", artifact, err)
		}
		if exists {
			return fmt.Errorf("adoption workspace %s already exists; rerun adopt to recover it", artifact)
		}
	}
	data, err := json.Marshal(transaction)
	if err != nil {
		return fmt.Errorf("encode adoption transaction: %w", err)
	}
	if err := writeFileAtomic(root, transaction.record, data); err != nil {
		return fmt.Errorf("record adoption transaction: %w", err)
	}
	return nil
}

func recoverTransaction(root *os.Root, source, destination string) error {
	expected := newTransaction(source, destination, false)
	transaction, found, err := readTransaction(root, expected)
	if err != nil || !found {
		return err
	}

	sourceInfo, sourceExists, err := lstat(root, transaction.Source)
	if err != nil {
		return fmt.Errorf("inspect interrupted source %s: %w", transaction.Source, err)
	}
	_, sourceBackupExists, err := lstat(root, transaction.sourceBackup)
	if err != nil {
		return fmt.Errorf("inspect interrupted source backup %s: %w", transaction.sourceBackup, err)
	}
	_, destinationBackupExists, err := lstat(root, transaction.destinationBackup)
	if err != nil {
		return fmt.Errorf("inspect interrupted destination backup %s: %w", transaction.destinationBackup, err)
	}

	if sourceExists && sourceLinksToDestination(root, transaction.Source, transaction.Destination) {
		if err := validateManagedArtifact(root, transaction.Destination); err == nil {
			return finishTransaction(root, transaction)
		} else if !sourceBackupExists {
			return fmt.Errorf("validate interrupted destination %s: %w", transaction.Destination, err)
		}
	}
	if !sourceExists {
		if !sourceBackupExists {
			return fmt.Errorf("interrupted adoption lost source %s and its backup %s", transaction.Source, transaction.sourceBackup)
		}
		return rollbackTransaction(root, transaction, destinationBackupExists)
	}
	if sourceBackupExists {
		return fmt.Errorf("interrupted adoption has both source %s and backup %s", transaction.Source, transaction.sourceBackup)
	}
	if destinationBackupExists {
		return fmt.Errorf("interrupted adoption has destination backup %s without source backup", transaction.destinationBackup)
	}
	if sourceInfo == nil {
		return fmt.Errorf("inspect interrupted source %s: no file info", transaction.Source)
	}
	if err := removeArtifact(root, transaction.stage); err != nil {
		return fmt.Errorf("remove abandoned adoption stage %s: %w", transaction.stage, err)
	}
	if err := removeTransaction(root, transaction); err != nil {
		return err
	}
	return nil
}

func rollbackTransaction(root *os.Root, transaction transaction, destinationBackupExists bool) error {
	_, destinationExists, err := lstat(root, transaction.Destination)
	if err != nil {
		return fmt.Errorf("inspect interrupted destination %s: %w", transaction.Destination, err)
	}
	if destinationBackupExists {
		if destinationExists {
			if err := removeArtifact(root, transaction.Destination); err != nil {
				return fmt.Errorf("remove interrupted destination %s: %w", transaction.Destination, err)
			}
		}
		if err := root.Rename(native(transaction.destinationBackup), native(transaction.Destination)); err != nil {
			return fmt.Errorf("restore interrupted destination %s: %w", transaction.Destination, err)
		}
	} else if !transaction.Overwrite && destinationExists {
		if err := removeArtifact(root, transaction.Destination); err != nil {
			return fmt.Errorf("remove interrupted destination %s: %w", transaction.Destination, err)
		}
	}
	if err := removeArtifact(root, transaction.stage); err != nil {
		return fmt.Errorf("remove interrupted stage %s: %w", transaction.stage, err)
	}
	if err := root.Rename(native(transaction.sourceBackup), native(transaction.Source)); err != nil {
		return fmt.Errorf("restore interrupted source %s: %w", transaction.Source, err)
	}
	if err := removeTransaction(root, transaction); err != nil {
		return err
	}
	return nil
}

func finishTransaction(root *os.Root, transaction transaction) error {
	for _, artifact := range []string{transaction.stage, transaction.sourceBackup, transaction.destinationBackup} {
		if err := removeArtifact(root, artifact); err != nil {
			return fmt.Errorf("clean completed adoption artifact %s: %w", artifact, err)
		}
	}
	return removeTransaction(root, transaction)
}

func readTransaction(root *os.Root, expected transaction) (transaction, bool, error) {
	info, exists, err := lstat(root, expected.record)
	if err != nil || !exists {
		return transaction{}, exists, err
	}
	if info == nil || !info.Mode().IsRegular() {
		return transaction{}, false, fmt.Errorf("adoption transaction %s is not a regular file", expected.record)
	}
	if info.Size() > maxTransactionSize {
		return transaction{}, false, fmt.Errorf("adoption transaction %s is %d bytes; limit is %d", expected.record, info.Size(), maxTransactionSize)
	}
	data, err := root.ReadFile(native(expected.record))
	if err != nil {
		return transaction{}, false, fmt.Errorf("read adoption transaction %s: %w", expected.record, err)
	}
	var stored transaction
	if err := json.Unmarshal(data, &stored); err != nil {
		return transaction{}, false, fmt.Errorf("decode adoption transaction %s: %w", expected.record, err)
	}
	if stored.Version != transactionVersion || stored.Source != expected.Source || stored.Destination != expected.Destination {
		return transaction{}, false, fmt.Errorf("adoption transaction %s does not match requested paths", expected.record)
	}
	stored = newTransaction(stored.Source, stored.Destination, stored.Overwrite)
	return stored, true, nil
}

func removeTransaction(root *os.Root, transaction transaction) error {
	info, exists, err := lstat(root, transaction.record)
	if err != nil || !exists {
		return err
	}
	if info == nil || !info.Mode().IsRegular() {
		return fmt.Errorf("adoption transaction %s is not a regular file", transaction.record)
	}
	if err := root.Remove(native(transaction.record)); err != nil {
		return fmt.Errorf("remove adoption transaction %s: %w", transaction.record, err)
	}
	return nil
}

func writeFileAtomic(root *os.Root, filePath string, data []byte) (err error) {
	if err := root.MkdirAll(native(path.Dir(filePath)), 0o750); err != nil {
		return fmt.Errorf("create parent for %s: %w", filePath, err)
	}
	temporary, err := temporaryPath(root, path.Dir(filePath), ".agentlink-write-")
	if err != nil {
		return err
	}
	file, err := root.OpenFile(native(temporary), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create temporary file for %s: %w", filePath, err)
	}
	defer func() {
		_ = file.Close()
		if err != nil {
			_ = root.Remove(native(temporary))
		}
	}()
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write temporary file for %s: %w", filePath, err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync temporary file for %s: %w", filePath, err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temporary file for %s: %w", filePath, err)
	}
	if err := root.Rename(native(temporary), native(filePath)); err != nil {
		return fmt.Errorf("publish adoption transaction %s: %w", filePath, err)
	}
	return nil
}

func removeArtifact(root *os.Root, artifact string) error {
	info, exists, err := lstat(root, artifact)
	if err != nil || !exists {
		return err
	}
	if info == nil || info.Mode()&fs.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
		return fmt.Errorf("adoption artifact %s is not a regular file or directory", artifact)
	}
	return root.RemoveAll(native(artifact))
}

func lstat(root *os.Root, artifact string) (fs.FileInfo, bool, error) {
	info, err := root.Lstat(native(artifact))
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return info, true, nil
}

func sourceLinksToDestination(root *os.Root, source, destination string) bool {
	info, exists, err := lstat(root, source)
	if err != nil || !exists || info == nil || info.Mode()&fs.ModeSymlink == 0 {
		return false
	}
	link, err := root.Readlink(native(source))
	return err == nil && linkResolvesTo(source, link, destination)
}

func validateManagedArtifact(root *os.Root, artifact string) error {
	info, exists, err := lstat(root, artifact)
	if err != nil {
		return fmt.Errorf("inspect managed artifact %s: %w", artifact, err)
	}
	if !exists {
		return fmt.Errorf("managed artifact %s: %w", artifact, os.ErrNotExist)
	}
	if info == nil || info.Mode()&fs.ModeSymlink != 0 || (!info.IsDir() && !info.Mode().IsRegular()) {
		return fmt.Errorf("managed artifact %s is not a regular file or directory", artifact)
	}
	return nil
}
