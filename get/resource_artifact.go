package get

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/hashicorp/go-getter/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceArtifact() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceArtifactCreate,
		ReadContext:   resourceArtifactRead,
		UpdateContext: resourceArtifactUpdate,
		DeleteContext: resourceArtifactDelete,

		CustomizeDiff: resourceArtifactCustomizeDiff,

		Schema: map[string]*schema.Schema{
			"archive": {
				Type:        schema.TypeString,
				Description: "configure explicit unarchiving behavior",
				Optional:    true,
				ForceNew:    true,
			},
			"required": {
				Type:        schema.TypeBool,
				Description: "ensure destination is always present",
				Optional:    true,
				Default:     false,
			},
			"checksum": {
				Type:        schema.TypeString,
				Description: "configure artifact checksumming",
				Optional:    true,
				ForceNew:    true,
			},
			"dest": {
				Type:        schema.TypeString,
				Description: "destination path",
				Required:    true,
				ForceNew:    true,
			},
			"insecure": {
				Type:        schema.TypeBool,
				Description: "disable TLS verification",
				Optional:    true,
			},
			"mode": {
				Type:             schema.TypeString,
				Description:      "get mode (any, file, dir)",
				Optional:         true,
				Default:          "any",
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"any", "dir", "file"}, false)),
			},
			"sum": {
				Type:        schema.TypeString,
				Description: "artifact checksum",
				Computed:    true,
			},
			"sum64": {
				Type:        schema.TypeString,
				Description: "base64 encoded artifact checksum",
				Computed:    true,
			},
			"url": {
				Type:        schema.TypeString,
				Description: "path to artifact (go-getter url)",
				Required:    true,
			},
			"workdir": {
				Type:        schema.TypeString,
				Description: "working directory",
				Optional:    true,
				ForceNew:    true,
			},
		},
	}
}

func resourceArtifactCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	req, err := resourceArtifactRequest(d)
	if err != nil {
		return diag.FromErr(err)
	}

	client := m.(*getter.Client)

	_, err = client.Get(ctx, req)
	if err != nil {
		return diag.Errorf("error getting url: %v", err)
	}

	req, err = resourceArtifactRequest(d)
	if err != nil {
		return diag.FromErr(err)
	}

	checksum, err := client.GetChecksum(ctx, req)
	if err != nil {
		return diag.Errorf("error getting checksum: %v", err)
	}
	if sum, sum64, ok := resourceArtifactSum(checksum); ok {
		d.Set("sum", sum)
		d.Set("sum64", sum64)
		d.SetId(sum)
	} else {
		sha256.New().Sum([]byte(req.Src))
		d.SetId(base64.RawStdEncoding.EncodeToString(sha256.New().Sum([]byte(req.Src))))
	}

	return resourceArtifactRead(ctx, d, m)
}

func resourceArtifactRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	return nil
}

func resourceArtifactUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	if _, always := d.GetChange("required"); always.(bool) {
		if _, err := os.Stat(d.Get("dest").(string)); err != nil {
			return resourceArtifactCreate(ctx, d, m)
		}
	}
	return nil
}

func resourceArtifactDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	dest := d.Get("dest").(string)
	if _, err := os.Stat(dest); err == nil {
		if err := os.Remove(dest); err != nil {
			return diag.FromErr(err)
		}
	}
	return nil
}

// resourceArtifactCustomizeDiff will ensure update when remote artifact checksum has changed
func resourceArtifactCustomizeDiff(ctx context.Context, d *schema.ResourceDiff, m interface{}) error {
	_, alwaysiface := d.GetChange("required")
	always := alwaysiface.(bool)
	if always {
		if _, err := os.Stat(d.Get("dest").(string)); err != nil {
			d.ForceNew("required")
			return nil
		}
	}

	req, err := resourceArtifactRequest(d)
	if err != nil {
		return err
	}

	checksum, err := m.(*getter.Client).GetChecksum(ctx, req)
	if err != nil {
		return fmt.Errorf("error getting checksum: %v", err)
	}

	prev, hasPrevious := d.GetOk("sum")
	if sum, _, ok := resourceArtifactSum(checksum); ok && (!hasPrevious || prev != sum) {
		d.ForceNew("sum")
		d.ForceNew("sum64")
	}

	return nil
}

// configProvider is a simplified abstraction of a ResourceData or ResourceDiff value
type configProvider interface {
	Get(string) interface{}
	GetOk(string) (interface{}, bool)
}

// resourceArtifactRequest generates a go-getter request from a ResourceData or ResourceDiff value
func resourceArtifactRequest(d configProvider) (*getter.Request, error) {
	req := &getter.Request{
		Dst: d.Get("dest").(string),
	}

	switch d.Get("mode").(string) {
	case "any":
		req.GetMode = getter.ModeAny
	case "dir":
		req.GetMode = getter.ModeDir
	case "file":
		req.GetMode = getter.ModeFile
	default:
		return nil, fmt.Errorf("expected mode to be one of [any, dir, file], got: %s", d.Get("mode").(string))
	}

	src, err := url.Parse(d.Get("url").(string))
	if err != nil {
		return nil, fmt.Errorf("error parsing url: %v", err)
	}
	params := src.Query()

	if archive, ok := d.GetOk("archive"); ok {
		params.Set("archive", archive.(string))
	}
	if checksum, ok := d.GetOk("checksum"); ok {
		params.Set("checksum", checksum.(string))
	}
	if insecure := d.Get("insecure").(bool); insecure {
		params.Set("insecure", fmt.Sprintf("%v", insecure))
	}
	src.RawQuery = params.Encode()
	req.Src = src.String()

	if pwd, ok := d.GetOk("workdir"); ok {
		req.Pwd = pwd.(string)
	}
	return req, nil
}

// resourceArtifactSum extracts the hex and base64 formatted sum from a FileChecksum value
func resourceArtifactSum(checksum *getter.FileChecksum) (string, string, bool) {
	if checksum == nil {
		return "", "", false
	}
	sum := checksum.String()
	sumSegments := strings.SplitN(sum, ":", 2)
	if len(sumSegments) != 2 {
		return sum, "", true
	}

	raw, err := hex.DecodeString(sumSegments[1])
	if err != nil {
		return sum, "", true
	}

	return sum, base64.StdEncoding.EncodeToString(raw), true
}
