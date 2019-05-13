package tests

import (
	"fmt"
	"github.com/nuclio/logger"
	"github.com/stretchr/testify/suite"
	"github.com/v3io/xcp/backends"
	"github.com/v3io/xcp/common"
	"github.com/v3io/xcp/operators"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var log logger.Logger
var tempdir string
var AWS_TEST_BUCKET string

var dummyContent = []byte("dummy content")

type testLocalBackend struct {
	suite.Suite
}

func (suite *testLocalBackend) SetupSuite() {
	AWS_TEST_BUCKET = os.Getenv("AWS_TEST_BUCKET")
	src, err := common.UrlParse(tempdir)
	suite.Require().Nil(err)

	client, err := backends.NewLocalClient(log, src)
	suite.Require().Nil(err)

	w, err := client.Writer(filepath.Join(tempdir, "a.txt"), nil)
	suite.Require().Nil(err)
	n, err := w.Write(dummyContent)
	suite.Require().Nil(err)
	suite.Require().Equal(n, len(dummyContent))

	opts := backends.FileMeta{
		Mtime: time.Now().Add(-23 * time.Hour),
		Mode:  777}
	w, err = client.Writer(filepath.Join(tempdir, "a.csv"), &opts)
	suite.Require().Nil(err)
	n, err = w.Write(dummyContent)
	suite.Require().Nil(err)
	suite.Require().Equal(n, len(dummyContent))
	err = w.Close()
	suite.Require().Nil(err)
}

func (suite *testLocalBackend) TestRead() {
	src, err := common.UrlParse(tempdir)
	suite.Require().Nil(err)

	client, err := backends.NewLocalClient(log, src)
	suite.Require().Nil(err)

	r, err := client.Reader(filepath.Join(tempdir, "a.csv"))
	data := make([]byte, 100)
	n, err := r.Read(data)
	suite.Require().Nil(err)
	suite.Require().Equal(n, len(dummyContent))
	suite.Require().Equal(data[0:n], dummyContent)
}

func (suite *testLocalBackend) TestList() {

	src, err := common.UrlParse(tempdir)
	listTask := backends.ListDirTask{Source: src, Filter: "*.*"}
	iter, err := operators.ListDir(&listTask, log)
	suite.Require().Nil(err)

	for iter.Next() {
		fmt.Printf("File %s: %+v", iter.Name(), iter.At())
	}
	suite.Require().Nil(iter.Err())
	summary := iter.Summary()
	fmt.Printf("Total files: %d,  Total size: %d\n",
		summary.TotalFiles, summary.TotalBytes)
	suite.Require().Equal(summary.TotalFiles, 2)

	listTask = backends.ListDirTask{Source: src, Filter: "*.csv"}
	iter, err = operators.ListDir(&listTask, log)
	suite.Require().Nil(err)
	_, err = iter.ReadAll()
	suite.Require().Nil(err)
	summary = iter.Summary()

	fmt.Printf("Total files: %d,  Total size: %d\n",
		summary.TotalFiles, summary.TotalBytes)
	suite.Require().Equal(summary.TotalFiles, 1)
}

func (suite *testLocalBackend) TestCopyToS3() {
	src, err := common.UrlParse(tempdir)
	suite.Require().Nil(err)

	listTask := backends.ListDirTask{Source: src, Filter: "*.*", WithMeta: true}
	dst, err := common.UrlParse("s3://" + AWS_TEST_BUCKET + "/xcptests")
	suite.Require().Nil(err)

	err = operators.CopyDir(&listTask, dst, log, 1)
	suite.Require().Nil(err)

	// read list dir content from S3
	listTask = backends.ListDirTask{Source: dst, Filter: "*.*"}
	dstdir, err := ioutil.TempDir("", "xcptest-dst")
	suite.Require().Nil(err)
	newdst, err := common.UrlParse(dstdir)
	suite.Require().Nil(err)

	err = operators.CopyDir(&listTask, newdst, log, 1)
	suite.Require().Nil(err)
}

func TestLocalBackendSuite(t *testing.T) {
	log, _ = common.NewLogger("debug")
	var err error
	tempdir, err = ioutil.TempDir("", "xcptest")
	//tempdir = "../tst1"
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("Target dir: ", tempdir)

	os.RemoveAll(tempdir) // clean up
	suite.Run(t, new(testLocalBackend))
	os.RemoveAll(tempdir)
}
