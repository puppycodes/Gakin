/*
	Gakin IRC Webhook bot
*/
package main

import (
	"fmt"
	"net/url"
	"net/http"
	"strconv"
	"io/ioutil"
	"github.com/jeffail/gabs"
	irc "github.com/fluffle/goirc/client"
)

var message = make(chan string);

func GitioShort(_url string) (string) {
	resp, err  := http.PostForm("http://git.io", url.Values{"url": {_url}});
	if err != nil {
		fmt.Printf("[*] GitioShort error: %s\n", err.Error());
	}
	defer resp.Body.Close();
	return resp.Header.Get("Location");
}

// TODO: Properly get commit count
func PushEvent(data *gabs.Container) {
	repo, _ := data.Search("repository", "full_name").Data().(string);
	user, _ := data.Search("pusher", "name").Data().(string);
	gitio := GitioShort(data.Search("head_commit", "url").Data().(string));
	commits, _ := data.Search("commits").Children();

	cobj, _ := data.Search("commits").ChildrenMap();

	commitlen := strconv.Itoa(len(cobj));

	message <- "[" + repo + "] " + user + " pushed " + commitlen + " commits " + gitio;


	for _, commit := range commits {
		hash := commit.Search("id").Data().(string)[0:6];
		msg := commit.Search("message").Data().(string);
		user := commit.Search("author", "name").Data().(string);

		message <- "[" + repo + "] " + user + " "  + hash + " - " + msg;
	}
}

func IssuesEvent(data *gabs.Container) {
	action := data.Search("action").Data().(string);

	repo, _ := data.Search("repository", "full_name").Data().(string);
	user, _ := data.Search("issue", "user", "login").Data().(string);
	title, _ := data.Search("issue", "title").Data().(string);
	inum, _ := data.Search("issue", "id").Data().(int);
	numstr := strconv.Itoa(inum);

	gitio := GitioShort(data.Search("issue", "html_url").Data().(string));

	switch action {
	case "opened":
		message <- "[" + repo + "] " + user + " opened issue #" + numstr + "\"" + title + "\"" + gitio;
	case "closed":
		message <- "[" + repo + "] " + user + " closed issue #" + numstr + "\"" + title + "\"" + gitio;
	case "reopened":
		message <- "[" + repo + "] " + user + " reopened issue #" + numstr + "\"" + title + "\"" + gitio;
	case "assigned":
		assignee,_ := data.Search("issue", "assignee", "login").Data().(string);
		message <- "[" + repo + "] " + user + " assigned issue #" + numstr + "\"" + title + "\" to " + assignee + " " + gitio;
	default:
		// Ignore it
	}
}

func ProcessEvent(data *gabs.Container, event string) {
	switch event {
	case "push":
		PushEvent(data);
	case "issues":
		IssuesEvent(data);
	default:
		fmt.Printf("[*] HOOK %s\n", event);
	}
}

func HandlePost(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "POST":
		body, err := ioutil.ReadAll(req.Body);
		if err != nil {
			fmt.Printf("[*] HandlePost error: %s\n", err.Error());
		}
		r, err := gabs.ParseJSON(body);
		if err != nil {
			fmt.Printf("[*] HandlePost error: %s\n", err.Error());
		}
		ProcessEvent(r, req.Header.Get("X-Github-Event"));
	default:
		w.Header().Set("Content-Type", "text/html");
		fmt.Fprintf(w, "<center><h1>Shhhhhhhhhhhhhhh</h1><br /><object width='800' height='600' data='http://archive.bad-alloc.net/other/flash/garrett.swf'></object></center>");
	}
}

func main() {
	http.HandleFunc("/", HandlePost);
	go IRCConnection("chat.freenode.net", "##XAMPP");

	http.ListenAndServe(":9987", nil);
}



func IRCConnection(host string, channel string) {
	IRCConnQuit := make(chan bool);
	run := true;
	cfg := irc.NewConfig("Gakin", "Gakin");

    cfg.Server = host;
    cfg.NewNick = func(n string) string { return n + "~" };

    cli := irc.Client(cfg);

    cli.EnableStateTracking();

	cli.HandleFunc(irc.DISCONNECTED, func(conn *irc.Conn, line *irc.Line) {
		fmt.Printf("[*] Connect Done\n");
		IRCConnQuit <- true;
		run = false;

	});
	cli.HandleFunc(irc.CONNECTED, func(conn *irc.Conn, line *irc.Line) {
		fmt.Printf("[*] Joining %s\n", channel);
		cli.Join(channel);
	})

	fmt.Printf("[*] Connecting to %s\n", host);
	if err := cli.Connect(); err != nil {
		fmt.Printf("[*] Connection error: %s\n", err.Error());
	}

	// Run Worker
	for run {
		cli.Privmsg(channel, <- message);
	}

	<- IRCConnQuit;
}
