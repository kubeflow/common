package labels

import (
	"testing"

	v1 "github.com/kubeflow/common/pkg/apis/common/v1"
)

func TestReplicaIndex(t *testing.T) {
	cases := map[string]struct {
		labels  map[string]string
		want    int
		wantErr bool
	}{
		"new": {
			labels: map[string]string{
				v1.ReplicaIndexLabel: "2",
			},
			want: 2,
		},
		"old": {
			labels: map[string]string{
				v1.ReplicaIndexLabelDeprecated: "3",
			},
			want: 3,
		},
		"none": {
			labels:  map[string]string{},
			wantErr: true,
		},
		"both": {
			labels: map[string]string{
				v1.ReplicaIndexLabel:           "4",
				v1.ReplicaIndexLabelDeprecated: "5",
			},
			want: 4,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := ReplicaIndex(tc.labels)
			if gotErr := err != nil; tc.wantErr != gotErr {
				t.Errorf("ReplicaIndex returned error (%t) want (%t)", gotErr, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("ReplicaIndex returned %d, want %d", got, tc.want)
			}
		})
	}
}

func TestReplicaType(t *testing.T) {
	cases := map[string]struct {
		labels  map[string]string
		want    string
		wantErr bool
	}{
		"new": {
			labels: map[string]string{
				v1.ReplicaTypeLabel: "Foo",
			},
			want: "Foo",
		},
		"old": {
			labels: map[string]string{
				v1.ReplicaTypeLabelDeprecated: "Bar",
			},
			want: "Bar",
		},
		"none": {
			labels:  map[string]string{},
			wantErr: true,
		},
		"both": {
			labels: map[string]string{
				v1.ReplicaTypeLabel:           "Baz",
				v1.ReplicaTypeLabelDeprecated: "Foo",
			},
			want: "Baz",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := ReplicaType(tc.labels)
			if gotErr := err != nil; tc.wantErr != gotErr {
				t.Errorf("ReplicaType returned error (%t) want (%t)", gotErr, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("ReplicaType returned %v, want %v", got, tc.want)
			}
		})
	}
}
