package uuidutil_test

import (
	"testing"

	"github.com/HallyG/fingrab/internal/util/uuidutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUnmarshallJSONUUID(t *testing.T) {
	tests := []struct {
		name         string
		data         []byte
		expectError  bool
		expectedUUID uuid.UUID
	}{
		{
			name:         "Valid UUID",
			data:         []byte("550e8400-e29b-41d4-a716-446655440000"),
			expectError:  false,
			expectedUUID: uuid.Must(uuid.Parse("550e8400-e29b-41d4-a716-446655440000")),
		},
		{
			name:         "Invalid UUID (wrong format)",
			data:         []byte("550e8400-e29b-41d4-a716-xyz"),
			expectError:  true,
			expectedUUID: uuid.Nil,
		},
		{
			name:         "Empty UUID string",
			data:         []byte(""),
			expectError:  true,
			expectedUUID: uuid.Nil,
		},
		{
			name:         "Valid UUID with leading/trailing spaces",
			data:         []byte(" 550e8400-e29b-41d4-a716-446655440000 "),
			expectError:  false,
			expectedUUID: uuid.Must(uuid.Parse("550e8400-e29b-41d4-a716-446655440000")),
		},
		{
			name:         "UUID with extra characters",
			data:         []byte("550e8400-e29b-41d4-a716-446655440000XYZ"),
			expectError:  true,
			expectedUUID: uuid.Nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := uuidutil.UnmarshallJSONUUID(test.data)

			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, test.expectedUUID, got)
		})
	}
}
