package interopsmoke

import "testing"

func TestValidateInvalidMessageOptions(t *testing.T) {
	for _, tc := range []struct {
		name               string
		blocks, txPerBlock uint
		wantErr            bool
	}{
		{"valid", 2, 3, false},
		{"zero blocks", 0, 1, true},
		{"zero tx per block", 1, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateInvalidMessageOptions(tc.blocks, tc.txPerBlock); (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestInvalidDirections(t *testing.T) {
	env := &smokeEnv{
		userA: &remoteUser{chain: &remoteChain{name: "L2A"}},
		userB: &remoteUser{chain: &remoteChain{name: "L2B"}},
	}
	for _, tc := range []struct {
		direction string
		wantNames []string
		wantErr   bool
	}{
		{direction: directionBoth, wantNames: []string{"A->B", "B->A"}},
		{direction: "", wantNames: []string{"A->B", "B->A"}},
		{direction: directionAToB, wantNames: []string{"A->B"}},
		{direction: directionBToA, wantNames: []string{"B->A"}},
		{direction: "sideways", wantErr: true},
	} {
		t.Run(tc.direction, func(t *testing.T) {
			env.direction = tc.direction
			dirs, err := invalidDirections(env)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if len(dirs) != len(tc.wantNames) {
				t.Fatalf("got %d directions, want %d", len(dirs), len(tc.wantNames))
			}
			for i, want := range tc.wantNames {
				if dirs[i].name != want {
					t.Fatalf("direction %d = %s, want %s", i, dirs[i].name, want)
				}
			}
		})
	}
}
