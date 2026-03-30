package hook

import (
	"errors"
	"testing"
)

func TestHook_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		hook    Hook
		wantErr error
	}{
		{
			name: "valid_upload_hook",
			hook: Hook{
				Name:        "S3 Upload",
				Endpoint:    "/upload",
				Method:      "POST",
				Type:        TypeUpload,
				Destination: "/tmp/uploads",
			},
			wantErr: nil,
		},
		{
			name: "valid_command_hook",
			hook: Hook{
				Name:     "Image Resize",
				Endpoint: "/resize",
				Method:   "PUT",
				Type:     TypeCommand,
				CommandTemplate: &ExecConfig{
					Command: "magick",
					Args:    []string{"convert"},
				},
			},
			wantErr: nil,
		},
		{
			name: "missing_name_edge_case",
			hook: Hook{
				Name:     "",
				Endpoint: "/test",
				Method:   "POST",
				Type:     TypeUpload,
			},
			wantErr: ErrHookNameRequired,
		},
		{
			name: "invalid_type_boundary",
			hook: Hook{
				Name:     "Unknown Type",
				Endpoint: "/test",
				Method:   "POST",
				Type:     "invalid_type",
			},
			wantErr: ErrHookTypeRequired,
		},
		{
			name: "nil_command_template_edge_case",
			hook: Hook{
				Name:            "Empty Command",
				Endpoint:        "/exec",
				Method:          "POST",
				Type:            TypeCommand,
				CommandTemplate: nil,
			},
			wantErr: ErrHookCmdTemplateRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.hook.Validate()

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Validate() expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Validate() expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Validate() unexpected error: %v", err)
			}
		})
	}
}

func TestHook_Validate_Advanced(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		hook    Hook
		wantErr error
	}{
		{
			name: "command_missing_executable_string",
			hook: Hook{
				Name:     "No Command",
				Endpoint: "/exec",
				Method:   "POST",
				Type:     TypeCommand,
				CommandTemplate: &ExecConfig{
					Command: "", // Empty command string
					Args:    []string{"-v"},
				},
			},
			wantErr: ErrHookCmdTemplateCmdRequired,
		},
		{
			name: "upload_missing_destination",
			hook: Hook{
				Name:      "No Dest",
				Endpoint:  "/upload",
				Method:    "POST",
				Type:      TypeUpload,
				// Destination is empty
			},
			wantErr: ErrHookUploadDestRequired,
		},
		{
			name: "zero_value_fields_valid",
			hook: Hook{
				Name:        "Minimal Upload",
				Endpoint:    "/u",
				Method:      "PUT",
				Type:        TypeUpload,
				Destination: "/tmp",
				MaxSizeMB:   0, // Testing if 0 is treated as 'no limit' or just a valid int
			},
			wantErr: nil,
		},
		{
			name: "command_with_zero_timeout",
			hook: Hook{
				Name:     "Quick Command",
				Endpoint: "/run",
				Method:   "POST",
				Type:     TypeCommand,
				CommandTemplate: &ExecConfig{
					Command:        "ls",
					TimeoutSeconds: 0,
				},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.hook.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
