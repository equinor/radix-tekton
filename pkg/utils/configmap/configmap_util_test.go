package configmap

import "testing"

func Test_getFolderListAsString(t *testing.T) {
	type args struct {
		changedFolders []string
	}
	scenarios := []struct {
		name string
		args args
		want string
	}{
		{name: "regular single path", args: args{changedFolders: []string{"/path/a123/123"}}, want: "/path/a123/123"},
		{name: "root path", args: args{changedFolders: []string{"/"}}, want: "/"},
		{name: "multiple paths", args: args{changedFolders: []string{"abc/", "/a/d/g", "/", "."}}, want: "abc/,/a/d/g,/,."},
		{name: "with commas", args: args{changedFolders: []string{"abc,de", "a,b,g"}}, want: "abc\\,de,a\\,b\\,g"},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			if got := getFolderListAsString(scenario.args.changedFolders); got != scenario.want {
				t.Errorf("getFolderListAsString() = %v, want %v", got, scenario.want)
			}
		})
	}
}
