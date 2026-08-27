package ddsutil

// Cross-validation against the golden reference files in testdata/. Each
// file is regenerated with the Go port and compared byte-for-byte.

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func goldenFill(data []byte) {
	for i := range data {
		data[i] = byte(i % 251)
	}
}

func TestGoldenFiles(t *testing.T) {
	cases := []struct {
		name  string
		build func() (*Dds, error)
	}{
		{
			name: "a8r8g8b8.dds",
			build: func() (*Dds, error) {
				dds, err := NewD3D(NewD3dParams{Height: 4, Width: 4, Format: D3DFormatA8R8G8B8})
				if err != nil {
					return nil, err
				}
				goldenFill(dds.Data)
				return dds, nil
			},
		},
		{
			name: "dxt5_mip.dds",
			build: func() (*Dds, error) {
				dds, err := NewD3D(NewD3dParams{Height: 8, Width: 8, Format: D3DFormatDXT5, MipmapLevels: u32p(3)})
				if err != nil {
					return nil, err
				}
				goldenFill(dds.Data)
				return dds, nil
			},
		},
		{
			name: "bc1.dds",
			build: func() (*Dds, error) {
				dds, err := NewDXGI(NewDxgiParams{
					Height: 8, Width: 8, Format: DxgiFormatBC1_UNorm,
					ResourceDimension: D3D10ResourceDimensionTexture2D,
					AlphaMode:         AlphaModeUnknown,
				})
				if err != nil {
					return nil, err
				}
				goldenFill(dds.Data)
				return dds, nil
			},
		},
		{
			name: "rgba8.dds",
			build: func() (*Dds, error) {
				dds, err := NewDXGI(NewDxgiParams{
					Height: 8, Width: 8, Format: DxgiFormatR8G8B8A8_UNorm,
					ResourceDimension: D3D10ResourceDimensionTexture2D,
					AlphaMode:         AlphaModeUnknown,
				})
				if err != nil {
					return nil, err
				}
				goldenFill(dds.Data)
				return dds, nil
			},
		},
		{
			name: "cube.dds",
			build: func() (*Dds, error) {
				c2 := Caps2CUBEMAP | Caps2CUBEMAP_ALLFACES
				dds, err := NewDXGI(NewDxgiParams{
					Height: 4, Width: 4, Format: DxgiFormatR8G8B8A8_UNorm,
					ArrayLayers:       u32p(6),
					Caps2:             &c2,
					IsCubemap:         true,
					ResourceDimension: D3D10ResourceDimensionTexture2D,
					AlphaMode:         AlphaModeUnknown,
				})
				if err != nil {
					return nil, err
				}
				goldenFill(dds.Data)
				return dds, nil
			},
		},
		{
			name: "volume.dds",
			build: func() (*Dds, error) {
				dds, err := NewD3D(NewD3dParams{Height: 4, Width: 4, Depth: u32p(2), Format: D3DFormatA8R8G8B8})
				if err != nil {
					return nil, err
				}
				goldenFill(dds.Data)
				return dds, nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			golden, err := os.ReadFile(filepath.Join("testdata", tc.name))
			if err != nil {
				t.Skipf("golden file missing: %v", err)
			}
			dds, err := tc.build()
			if err != nil {
				t.Fatalf("build failed: %v", err)
			}
			var got bytes.Buffer
			if err := dds.Write(&got); err != nil {
				t.Fatalf("Write failed: %v", err)
			}
			if !bytes.Equal(golden, got.Bytes()) {
				// Find first difference for the report.
				gb := got.Bytes()
				n := len(golden)
				if len(gb) < n {
					n = len(gb)
				}
				for i := 0; i < n; i++ {
					if golden[i] != gb[i] {
						t.Fatalf("byte %d differs: golden=0x%02x got=0x%02x (len golden=%d got=%d)",
							i, golden[i], gb[i], len(golden), len(gb))
					}
				}
				t.Fatalf("length differs: golden=%d got=%d", len(golden), len(gb))
			}

			// Also verify the golden file can be read back with sensible
			// fields.
			back, err := Read(bytes.NewReader(golden))
			if err != nil {
				t.Fatalf("Read(golden) failed: %v", err)
			}
			if back.GetWidth() != dds.GetWidth() || back.GetHeight() != dds.GetHeight() {
				t.Errorf("dims %dx%d, want %dx%d",
					back.GetWidth(), back.GetHeight(), dds.GetWidth(), dds.GetHeight())
			}
			if len(back.Data) != len(dds.Data) {
				t.Errorf("data len %d, want %d", len(back.Data), len(dds.Data))
			}
		})
	}
}
