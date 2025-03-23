package es

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		entity  Entity
		want    string
		wantErr bool
	}{
		{
			name:    "valid entity",
			entity:  Entity{ID: uuid.New(), Type: "test"},
			wantErr: false,
		},
		{
			name:    "nil ID",
			entity:  Entity{ID: uuid.Nil, Type: "test"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.entity.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				expected := fmt.Sprintf("\"%s:%s\"", tt.entity.Type, tt.entity.ID)
				assert.JSONEq(t, expected, string(got))
			}
		})
	}
}

func TestUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		want    Entity
		wantErr bool
	}{
		{
			name:    "valid entity",
			data:    "\"test:550e8400-e29b-41d4-a716-446655440000\"",
			want:    Entity{ID: uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), Type: "test"},
			wantErr: false,
		},
		{
			name:    "invalid format",
			data:    "\"invalid_format\"",
			wantErr: true,
		},
		{
			name:    "invalid UUID",
			data:    "\"test:invalid-uuid\"",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Entity
			err := got.UnmarshalJSON([]byte(tt.data))
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
