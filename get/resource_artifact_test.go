package get

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
)

func TestResourceArtifactRequest(t *testing.T) {
	cases := []struct {
		url      string
		archive  string
		checksum string
		insecure *bool
		mode     string
		expected string
	}{
		{
			url:      "foo/bar.zip",
			checksum: "md5:123",
			expected: "foo/bar.zip?checksum=md5%3A123",
		},
		{
			url:      "foo/bar.zip?checksum=file:123",
			checksum: "md5:123",
			expected: "foo/bar.zip?checksum=md5%3A123",
		},
		{
			url:      "file:///tmp/foo.zip?bar=qux",
			archive:  "false",
			insecure: testBool(true),
			expected: "file:///tmp/foo.zip?archive=false&bar=qux&insecure=true",
		},
	}

	for _, c := range cases {
		d := resourceArtifact().TestResourceData()

		d.Set("url", c.url)
		d.Set("dest", "/tmp")
		if c.archive != "" {
			d.Set("archive", c.archive)
		}
		if c.checksum != "" {
			d.Set("checksum", c.checksum)
		}
		if c.insecure != nil {
			d.Set("insecure", *c.insecure)
		}
		if c.mode != "" {
			d.Set("mode", c.mode)
		} else {
			d.Set("mode", "any")
		}

		req, err := resourceArtifactRequest(d)
		if !assert.NoError(t, err) {
			t.FailNow()
		}

		assert.Equal(t, c.expected, req.Src)
	}
}

func TestArtifact_localfile(t *testing.T) {
	// generate local file
	dir := t.TempDir()
	srcPath := fmt.Sprintf("%s/source.txt", dir)
	destPath := fmt.Sprintf("%s/dest.txt", dir)

	createConfig, createChecksum, createChecksum64, createFn := testLocalFile(t, srcPath, destPath, "abc")
	createFn()

	updateConfig, updateChecksum, updateChecksum64, updateFn := testLocalFile(t, srcPath, destPath, "123")

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		Providers: map[string]*schema.Provider{
			"get": Provider(),
		},
		Steps: []resource.TestStep{
			{
				Config: createConfig,
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						b, err := ioutil.ReadFile(destPath)
						assert.NoError(t, err)
						assert.Equal(t, "abc", string(b))
						return nil
					},
					resource.TestCheckResourceAttr("get_artifact.this", "sum", createChecksum),
					resource.TestCheckResourceAttr("get_artifact.this", "sum64", createChecksum64),
				),
			},
			{
				Config:    updateConfig,
				PreConfig: updateFn,
				Check: resource.ComposeTestCheckFunc(
					func(s *terraform.State) error {
						b, err := ioutil.ReadFile(destPath)
						assert.NoError(t, err)
						assert.Equal(t, "123", string(b))
						return nil
					},
					resource.TestCheckResourceAttr("get_artifact.this", "sum", updateChecksum),
					resource.TestCheckResourceAttr("get_artifact.this", "sum64", updateChecksum64),
				),
			},
		},
	})
}

func testBool(v bool) *bool {
	return &v
}

func testLocalFile(t *testing.T, src, dest, content string) (string, string, string, func()) {
	h := sha256.New()
	if _, err := h.Write([]byte(content)); err != nil {
		t.Fatalf("error computing hash of local file: %v", err)
	}

	sum := h.Sum(nil)

	checksum := fmt.Sprintf("sha256:%x", sum)
	checksum64 := base64.StdEncoding.EncodeToString(sum)

	config := fmt.Sprintf(`
resource "get_artifact" "this" {
	url      = "%s"
	checksum = "%s"
	dest     = "%s"
	mode     = "file"
}
`, src, checksum, dest)

	return config, checksum, checksum64, func() {
		if err := ioutil.WriteFile(src, []byte(content), 0777); err != nil {
			t.Fatalf("error writing local file: %v", err)
		}
	}
}
