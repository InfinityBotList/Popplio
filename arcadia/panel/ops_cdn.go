package panel

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"popplio/arcadia/cdnpath"
	"popplio/arcadia/impls"
	"popplio/arcadia/types"
	"popplio/config"
	"popplio/state"

	perms "github.com/infinitybotlist/kittycat/go"
	"go.uber.org/zap"
)

// hasCdnPerm delegates to the leaf package so the check is unit tested there.
func hasCdnPerm(userPerms []perms.Permission, scope, op string) bool {
	return cdnpath.HasPerm(userPerms, scope, op)
}

func validateName(name string) error { return cdnpath.ValidateName(name) }

func validatePath(p string) error { return cdnpath.ValidatePath(p) }

// cdnScopes returns the scope map for the current environment.
func cdnScopes() map[string]config.CdnScopeData {
	return state.Config.Arcadia.Panel.CdnScopes.Parse()
}

func (s *Server) uploadCdnFileChunk(ctx context.Context, q *types.QUploadCdnFileChunk) (response, error) {
	authData, err := checkAuth(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	userPerms, err := resolvedPerms(ctx, authData.UserID)

	if err != nil {
		return response{}, err
	}

	if !perms.HasPerm(userPerms, perms.PermissionFromString("cdn.upload_chunk")) {
		return writeText(http.StatusForbidden, "You do not have permission to upload chunks to the CDN right now [cdn.upload_chunk]"), nil
	}

	state.Logger.Info("Got chunk", zap.Int("len", len(q.Chunk)))

	if len(q.Chunk) > 100_000_000 {
		return writeText(http.StatusBadRequest, "Chunk size is too large"), nil
	}

	if len(q.Chunk) == 0 {
		return writeText(http.StatusBadRequest, "Chunk is empty"), nil
	}

	// QUIRK (reproduced): the chunk id is generated ONCE, outside the retry loop,
	// so the ten attempts all test the same id. See CONFORMANCE.md.
	chunkID := impls.GenRandom(32)

	for tries := 0; tries < 10; tries++ {
		if !s.chunks.Has(chunkID) {
			s.chunks.Put(chunkID, q.Chunk)

			return writeText(http.StatusOK, chunkID), nil
		}
	}

	return writeText(http.StatusInternalServerError, "Failed to generate a chunk ID"), nil
}

func (s *Server) listCdnScopes(ctx context.Context, q *types.QLoginTokenOnly) (response, error) {
	authData, err := checkAuth(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	userPerms, err := resolvedPerms(ctx, authData.UserID)

	if err != nil {
		return response{}, err
	}

	if !perms.HasPerm(userPerms, perms.PermissionFromString("cdn.list_scopes")) {
		return writeText(http.StatusForbidden, "You do not have permission to list the CDN's scopes right now [cdn.list_scopes]"), nil
	}

	return writeJSON(http.StatusOK, cdnScopes()), nil
}

func (s *Server) getMainCdnScope(ctx context.Context, q *types.QLoginTokenOnly) (response, error) {
	// No permission check.
	if _, err := checkAuth(ctx, q.LoginToken); err != nil {
		return response{}, err
	}

	return writeText(http.StatusOK, state.Config.Arcadia.Panel.MainScope), nil
}

func (s *Server) updateCdnAsset(ctx context.Context, q *types.QUpdateCdnAsset) (response, error) {
	authData, err := checkAuth(ctx, q.LoginToken)

	if err != nil {
		return response{}, err
	}

	userPerms, err := resolvedPerms(ctx, authData.UserID)

	if err != nil {
		return response{}, err
	}

	scope, ok := cdnScopes()[q.CdnScope]

	if !ok {
		return writeText(http.StatusBadRequest, "Invalid CDN scope"), nil
	}

	if err := validateName(q.Name); err != nil {
		return response{}, newError(err)
	}

	if err := validatePath(q.Path); err != nil {
		return response{}, newError(err)
	}

	assetPath := scope.Path
	if q.Path != "" {
		assetPath = scope.Path + "/" + q.Path
	}

	assetFinalPath := assetPath
	if q.Name != "" {
		assetFinalPath = assetPath + "/" + q.Name
	}

	// HARDENING (§14b): the validators above are pure string checks and are the
	// only thing standing between a staff member and the CDN root. This is a
	// second, structural containment check on the resolved paths.
	if !containedInScope(scope.Path, assetPath) || !containedInScope(scope.Path, assetFinalPath) {
		return writeText(http.StatusBadRequest, "Asset path cannot contain non-ASCII characters, dot-dots, doubleslashes, backslashes or start with a slash"), nil
	}

	switch {
	case q.Action.ListPath != nil:
		return s.cdnListPath(userPerms, q.CdnScope, scope, assetPath)
	case q.Action.ReadFile != nil:
		return s.cdnReadFile(userPerms, q.CdnScope, assetFinalPath)
	case q.Action.CreateFolder != nil:
		return s.cdnCreateFolder(userPerms, q.CdnScope, assetFinalPath)
	case q.Action.AddFile != nil:
		return s.cdnAddFile(userPerms, q.CdnScope, assetPath, assetFinalPath, q.Action.AddFile)
	case q.Action.CopyFile != nil:
		return s.cdnCopyFile(userPerms, q.CdnScope, scope, assetFinalPath, q.Action.CopyFile)
	case q.Action.Delete != nil:
		return s.cdnDelete(userPerms, q.CdnScope, assetFinalPath)
	case q.Action.PersistGit != nil:
		return s.cdnPersistGit(ctx, userPerms, q.CdnScope, assetFinalPath, q.Action.PersistGit)
	default:
		return response{}, errStatus(http.StatusBadRequest, "No CDN action was specified")
	}
}

func (s *Server) cdnListPath(userPerms []perms.Permission, scopeName string, scope config.CdnScopeData, assetPath string) (response, error) {
	if !hasCdnPerm(userPerms, scopeName, "list") {
		return writeText(http.StatusForbidden, "You do not have permission to list CDN assets right now [list]"), nil
	}

	meta, err := os.Stat(assetPath)

	if err != nil {
		return writeText(http.StatusBadRequest, "Fetching asset metadata failed: "+err.Error()), nil
	}

	if !meta.IsDir() {
		return writeText(http.StatusBadRequest, "Asset path already exists and is not a directory"), nil
	}

	entries, err := os.ReadDir(assetPath)

	if err != nil {
		return response{}, newError(err)
	}

	files := make([]types.CdnAssetItem, 0, len(entries))

	for _, entry := range entries {
		info, err := entry.Info()

		if err != nil {
			return response{}, newError(err)
		}

		files = append(files, types.CdnAssetItem{
			Name: entry.Name(),
			// The scope root prefix is stripped so the panel sees a relative path.
			Path:         strings.ReplaceAll(filepath.Join(assetPath, entry.Name()), scope.Path, ""),
			Size:         uint64(info.Size()),
			LastModified: uint64(info.ModTime().Unix()),
			IsDir:        info.IsDir(),
			Permissions:  uint32(info.Mode().Perm()),
		})
	}

	return writeJSON(http.StatusOK, files), nil
}

func (s *Server) cdnReadFile(userPerms []perms.Permission, scopeName, assetFinalPath string) (response, error) {
	if !hasCdnPerm(userPerms, scopeName, "read_file") {
		return writeText(http.StatusForbidden, "You do not have permission to read CDN files right now [read_file]"), nil
	}

	meta, err := os.Stat(assetFinalPath)

	if err != nil {
		return writeText(http.StatusBadRequest, "Fetching asset metadata failed: "+err.Error()), nil
	}

	if !meta.Mode().IsRegular() {
		return writeText(http.StatusBadRequest, "Asset path is not a file"), nil
	}

	file, err := os.Open(assetFinalPath)

	if err != nil {
		return writeText(http.StatusBadRequest, "Reading file failed: "+err.Error()), nil
	}

	// Streamed, never buffered: these files can be large.
	return writeStream(http.StatusOK, file), nil
}

func (s *Server) cdnCreateFolder(userPerms []perms.Permission, scopeName, assetFinalPath string) (response, error) {
	if !hasCdnPerm(userPerms, scopeName, "create_folder") {
		return writeText(http.StatusForbidden, "You do not have permission to create CDN folders right now [create_folder]"), nil
	}

	_, err := os.Stat(assetFinalPath)

	switch {
	case err == nil:
		return writeText(http.StatusBadRequest, "Asset path already exists"), nil
	case !errors.Is(err, fs.ErrNotExist):
		return writeText(http.StatusBadRequest, "Fetching asset metadata failed due to unknown error: "+err.Error()), nil
	}

	if err := os.MkdirAll(assetFinalPath, 0o755); err != nil {
		return response{}, newError(err)
	}

	return writeNoContent(), nil
}

func (s *Server) cdnAddFile(userPerms []perms.Permission, scopeName, assetPath, assetFinalPath string, action *types.CdnAddFile) (response, error) {
	if !hasCdnPerm(userPerms, scopeName, "add_file") {
		return writeText(http.StatusForbidden, "You do not have permission to add CDN files right now [cdn.add_file]"), nil
	}

	if len(action.Chunks) == 0 {
		return writeText(http.StatusBadRequest, "No chunks were provided"), nil
	}

	if len(action.Chunks) > 100_000 {
		return writeText(http.StatusBadRequest, "Too many chunks were provided"), nil
	}

	for _, chunk := range action.Chunks {
		if !s.chunks.Has(chunk) {
			return writeText(http.StatusBadRequest, "Chunk does not exist"), nil
		}
	}

	meta, err := os.Stat(assetFinalPath)

	switch {
	case err == nil:
		if !action.Overwrite {
			return writeText(http.StatusBadRequest, "Asset already exists"), nil
		}

		if meta.IsDir() {
			return writeText(http.StatusBadRequest, "Asset to be replaced is a directory"), nil
		}
	case !errors.Is(err, fs.ErrNotExist):
		return writeText(http.StatusBadRequest, "Fetching asset metadata failed due to unknown error: "+err.Error()), nil
	}

	dirMeta, err := os.Stat(assetPath)

	switch {
	case err == nil:
		if !dirMeta.IsDir() {
			return writeText(http.StatusBadRequest, "Asset path already exists and is not a directory"), nil
		}
	case errors.Is(err, fs.ErrNotExist):
		if err := os.MkdirAll(assetPath, 0o755); err != nil {
			return response{}, newError(err)
		}
	default:
		return writeText(http.StatusBadRequest, "Fetching asset metadata failed due to unknown error: "+err.Error()), nil
	}

	tmpPath := fmt.Sprintf("/tmp/arcadia-cdn-file%s@%s", impls.GenRandom(32), strings.ReplaceAll(assetFinalPath, "/", ">"))

	// BUG FIXED (flagged): upstream returns early on a hash mismatch without
	// removing the temp file, leaking it while the chunks it was built from have
	// already been consumed. The deferred cleanup below removes the temp file on
	// every exit path. This changes no response and no CDN content.
	defer func() {
		if err := os.Remove(tmpPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			state.Logger.Error("panel: failed removing CDN temp file", zap.Error(err), zap.String("path", tmpPath))
		}
	}()

	hash, err := s.assembleChunks(tmpPath, action.Chunks)

	if err != nil {
		return response{}, newError(err)
	}

	if action.SHA512 != hash {
		return writeText(http.StatusBadRequest, "SHA512 hash does not match"), nil
	}

	if err := copyFile(tmpPath, assetFinalPath); err != nil {
		return response{}, newError(err)
	}

	return writeNoContent(), nil
}

// assembleChunks writes the chunks in order to a temp file, fsyncs it and
// returns its lowercase-hex SHA-512. Each chunk is removed from the cache as it
// is consumed.
func (s *Server) assembleChunks(tmpPath string, chunks []string) (string, error) {
	tmp, err := os.Create(tmpPath)

	if err != nil {
		return "", err
	}

	defer tmp.Close()

	hasher := sha512.New()

	for _, id := range chunks {
		data, ok := s.chunks.Take(id)

		if !ok {
			return "", fmt.Errorf("Chunk %s does not exist", id)
		}

		if _, err := tmp.Write(data); err != nil {
			return "", err
		}

		hasher.Write(data)
	}

	if err := tmp.Sync(); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)

	if err != nil {
		return err
	}

	defer in.Close()

	out, err := os.Create(dst)

	if err != nil {
		return err
	}

	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}

func (s *Server) cdnCopyFile(userPerms []perms.Permission, scopeName string, scope config.CdnScopeData, assetFinalPath string, action *types.CdnCopyFile) (response, error) {
	if !hasCdnPerm(userPerms, scopeName, "copy_file") {
		return writeText(http.StatusForbidden, "You do not have permission to copy files right now [copy_file]"), nil
	}

	if err := validatePath(action.CopyTo); err != nil {
		return response{}, newError(err)
	}

	if action.CopyTo == "" {
		return writeText(http.StatusBadRequest, "copy_to location cannot be empty"), nil
	}

	copyTo := scope.Path + "/" + action.CopyTo

	if !containedInScope(scope.Path, copyTo) {
		return writeText(http.StatusBadRequest, "Asset path cannot contain non-ASCII characters, dot-dots, doubleslashes, backslashes or start with a slash"), nil
	}

	destMeta, err := os.Stat(copyTo)

	switch {
	case err == nil:
		if !destMeta.IsDir() && !action.Overwrite {
			return writeText(http.StatusBadRequest, "copy_to location already exists"), nil
		}
	case !errors.Is(err, fs.ErrNotExist):
		return writeText(http.StatusBadRequest, "Fetching asset metadata failed due to unknown error: "+err.Error()), nil
	}

	srcMeta, err := os.Lstat(assetFinalPath)

	if err != nil {
		return writeText(http.StatusBadRequest, "Could not find asset: "+err.Error()+fmt.Sprintf(" (path: %s)", assetFinalPath)), nil
	}

	switch {
	case srcMeta.Mode()&fs.ModeSymlink != 0 || srcMeta.Mode().IsRegular():
		if action.DeleteOriginal {
			if err := os.Rename(assetFinalPath, copyTo); err != nil {
				return response{}, newError(fmt.Errorf("Failed to rename file: %s from %s to %s", err, assetFinalPath, copyTo))
			}
		} else if err := copyFile(assetFinalPath, copyTo); err != nil {
			return response{}, newError(err)
		}
	case srcMeta.IsDir():
		if action.DeleteOriginal {
			if err := renameDirAll(assetFinalPath, copyTo); err != nil {
				return response{}, newError(err)
			}

			if err := os.RemoveAll(assetFinalPath); err != nil {
				return response{}, newError(err)
			}
		} else if err := copyDirAll(assetFinalPath, copyTo); err != nil {
			return response{}, newError(err)
		}
	}

	return writeNoContent(), nil
}

func renameDirAll(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)

	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := renameDirAll(srcPath, dstPath); err != nil {
				return err
			}

			continue
		}

		if err := os.Rename(srcPath, dstPath); err != nil {
			return err
		}
	}

	return nil
}

func copyDirAll(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)

	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDirAll(srcPath, dstPath); err != nil {
				return err
			}

			continue
		}

		if err := copyFile(srcPath, dstPath); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) cdnDelete(userPerms []perms.Permission, scopeName, assetFinalPath string) (response, error) {
	if !hasCdnPerm(userPerms, scopeName, "delete") {
		return writeText(http.StatusForbidden, "You do not have permission to delete CDN assets right now [delete]"), nil
	}

	meta, err := os.Lstat(assetFinalPath)

	if err != nil {
		return writeText(http.StatusBadRequest, "Could not find asset: "+err.Error()), nil
	}

	switch {
	case meta.Mode()&fs.ModeSymlink != 0 || meta.Mode().IsRegular():
		if err := os.Remove(assetFinalPath); err != nil {
			return response{}, newError(err)
		}
	case meta.IsDir():
		if err := os.RemoveAll(assetFinalPath); err != nil {
			return response{}, newError(err)
		}
	}

	return writeNoContent(), nil
}

// orderedMap preserves insertion order on the way out, which Go maps do not. The
// panel displays the PersistGit output verbatim, key order included.
type orderedMap struct {
	keys   []string
	values map[string]string
}

func newOrderedMap() *orderedMap {
	return &orderedMap{values: make(map[string]string)}
}

func (m *orderedMap) Insert(key, value string) {
	if _, exists := m.values[key]; !exists {
		m.keys = append(m.keys, key)
	}

	m.values[key] = value
}

func (m *orderedMap) MarshalJSON() ([]byte, error) {
	var buf strings.Builder

	buf.WriteByte('{')

	for i, key := range m.keys {
		if i > 0 {
			buf.WriteByte(',')
		}

		k, err := jsonString(key)

		if err != nil {
			return nil, err
		}

		v, err := jsonString(m.values[key])

		if err != nil {
			return nil, err
		}

		buf.Write(k)
		buf.WriteByte(':')
		buf.Write(v)
	}

	buf.WriteByte('}')

	return []byte(buf.String()), nil
}

func (s *Server) cdnPersistGit(ctx context.Context, userPerms []perms.Permission, scopeName, assetFinalPath string, action *types.CdnPersistGit) (response, error) {
	if !hasCdnPerm(userPerms, scopeName, "persist_git") {
		return writeText(http.StatusForbidden, "You do not have permission to persist CDN git right now [cdn.persist_git]"), nil
	}

	out := newOrderedMap()

	// SECURITY: the commit message is passed as its own argv element and never
	// through a shell.
	stdout, status, err := runGit(ctx, assetFinalPath, nil, "rev-parse", "--show-toplevel")

	if err != nil {
		return response{}, newError(fmt.Errorf("Failed to execute git rev-parse: %s", err))
	}

	repoRoot := strings.ReplaceAll(strings.TrimSpace(stdout), "\n", "")
	out.Insert("git rev-parse --show-toplevel", repoRoot)

	if status != "" {
		out.Insert("error", status)
		return writeJSON(http.StatusOK, out), nil
	}

	currDir := repoRoot
	if action.CurrentDir {
		currDir = assetFinalPath
	}

	out.Insert("[dir]", currDir)

	env := []string{"GIT_TERMINAL_PROMPT=0"}

	stdout, status, err = runGit(ctx, currDir, env, "add", ".")

	if err != nil {
		return response{}, newError(fmt.Errorf("Failed to execute git add: %s", err))
	}

	out.Insert("git add", strings.TrimSpace(stdout))

	if status != "" {
		out.Insert("error", status)
		return writeJSON(http.StatusOK, out), nil
	}

	stdout, status, err = runGit(ctx, currDir, env, "commit", "-m", action.Message)

	if err != nil {
		return response{}, newError(fmt.Errorf("Failed to execute git commit: %s", err))
	}

	out.Insert("git commit", strings.TrimSpace(stdout))

	// A failed commit does NOT return: it falls through to the push.
	if status != "" {
		out.Insert("error_gc", status)
	}

	stdout, status, err = runGit(ctx, currDir, env, "push", "--force")

	if err != nil {
		return response{}, newError(fmt.Errorf("Failed to execute git push: %s", err))
	}

	out.Insert("git push", strings.TrimSpace(stdout))

	if status != "" {
		out.Insert("error_gp", status)
		return writeJSON(http.StatusOK, out), nil
	}

	return writeJSON(http.StatusOK, out), nil
}

// runGit runs a git subcommand and returns its stdout plus a non-empty status
// string when the command failed. The status string matches Rust's
// ExitStatus::to_string().
func runGit(ctx context.Context, dir string, env []string, args ...string) (stdout string, status string, err error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}

	out, runErr := cmd.Output()
	stdout = string(out)

	if runErr != nil {
		var exitErr *exec.ExitError

		if errors.As(runErr, &exitErr) {
			return stdout, exitStatusString(exitErr), nil
		}

		return stdout, "", runErr
	}

	return stdout, "", nil
}

// exitStatusString mirrors Rust's Display for ExitStatus, e.g. "exit status: 1".
func exitStatusString(err *exec.ExitError) string {
	if code := err.ExitCode(); code >= 0 {
		return fmt.Sprintf("exit status: %d", code)
	}

	return err.String()
}
