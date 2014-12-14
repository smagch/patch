package patch_test

import (
	"fmt"
	"github.com/smagch/patch"
	"strings"
)

func ExamplePatcher_Unmarshal() {
	type User struct {
		ID     int
		Name   string
		Email  string
		Active bool
	}
	p := patch.New(User{})
	data, err := p.Unmarshal([]byte(`{"Name": "gopher", "Active": true}`))
	if err != nil {
		fmt.Println(err.Error())
	}
	keys := data.Keys()
	fmt.Println(strings.Join(keys, ","))
	// Output:
	// Name,Active
}

func ExampleFields_Remove() {
	type User struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	p := patch.New(User{})
	data, err := p.Unmarshal([]byte(`{"name": "gopher", "email": "golang@hoge.hoge"}`))
	if err != nil {
		fmt.Println(err.Error())
	}
	email, found := data.Get("email")
	if found {
		data.Remove("email")
	}
	fmt.Println(email)
	fmt.Printf("%v", data.Values())
	// Output:
	// golang@hoge.hoge
	// [gopher]
}

func ExampleFields_Set() {
	type Article struct {
		ID     int    `json:"id"`
		UserID int    `json:"user_id"`
		Title  string `json:"title"`
		Desc   string `json:"desc"`
		Body   string `json:"body"`
	}
	p := patch.New(Article{})
	f, err := p.Unmarshal([]byte(`{"title": "the gopher", "desc": "description"}`))
	if err != nil {
		fmt.Println(err.Error())
	}
	f.Set("desc", "goggles its eyes")
	f.Set("body", "with deadly pale body")
	fmt.Printf("%v", f.Values())
	// Output:
	// [the gopher goggles its eyes with deadly pale body]
}

func ExampleSQL_Query() {
	type Post struct {
		ID          int    `json:"id"`
		UserId      int    `json:"user_id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Body        string `json:"body"`
	}
	p := patch.New(Post{})
	data, err := p.Unmarshal([]byte(`{"title": "Space Gopher", "body": "The body"}`))
	if err != nil {
		fmt.Println(err.Error())
	}
	id := 947
	q, args := data.SQL().Query(id)
	fmt.Println(q)
	query := fmt.Sprintf(`UPDATE posts SET %s WHERE id = ?`, q)
	fmt.Println(query)
	fmt.Printf("%#v", args)
	// Output:
	// title=?,body=?
	// UPDATE posts SET title=?,body=? WHERE id = ?
	// []interface {}{"Space Gopher", "The body", 947}
}

func ExampleSQL_QueryPostgres() {
	type Post struct {
		ID          int    `json:"id"`
		UserId      int    `json:"user_id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Body        string `json:"body"`
	}
	p := patch.New(Post{})
	data, err := p.Unmarshal([]byte(`{"title": "Space Gopher", "body": "The body"}`))
	if err != nil {
		fmt.Println(err.Error())
	}
	id := 947
	q, args := data.SQL().QueryPostgres(id)
	fmt.Println(q)
	query := fmt.Sprintf(`UPDATE posts SET %s WHERE id = $1`, q)
	fmt.Println(query)
	fmt.Printf("%#v", args)
	// Output:
	// title=$2,body=$3
	// UPDATE posts SET title=$2,body=$3 WHERE id = $1
	// []interface {}{947, "Space Gopher", "The body"}
}
