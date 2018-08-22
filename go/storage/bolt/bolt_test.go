package bolt

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/oasislabs/ekiden/go/epochtime/mock"
	"github.com/oasislabs/ekiden/go/storage/internal/tester"
)

func TestStorageBolt(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "ekiden-storage-bolt-test")
	require.NoError(t, err, "TempDir()")
	defer os.RemoveAll(tmpDir)

	timeSource := mock.New()
	backend, err := New(filepath.Join(tmpDir, DBFile), timeSource)
	require.NoError(t, err, "New()")
	defer backend.Cleanup()

	tester.StorageImplementationTest(t, backend, timeSource)
}