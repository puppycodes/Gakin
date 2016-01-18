/*
	Gakin IRC Webhook bot
*/
package main

import (
	"os"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"net/http"
	"io/ioutil"
	"math/rand"
	"github.com/jeffail/gabs"
	irc "github.com/fluffle/goirc/client"
)

var message = make(chan string);
var sauce_key = "";

var messages map[string]string;


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
	repo, _ := data.S("repository", "full_name").Data().(string);
	user, _ := data.S("pusher", "name").Data().(string);
	gitio := GitioShort(data.S("head_commit", "url").Data().(string));
	commits, _ := data.S("commits").Children();

	cobj, _ := data.S("commits").ChildrenMap();
	fmt.Printf("[!] Commit # %d\n", len(cobj));
	commitlen := strconv.Itoa(len(cobj));

	message <- "[" + repo + "] " + user + " pushed " + commitlen + " commits " + gitio;


	for _, commit := range commits {
		hash := commit.S("id").Data().(string)[0:6];
		msg := commit.S("message").Data().(string);
		user := commit.S("author", "name").Data().(string);

		message <- "[" + repo + "] " + user + " "  + hash + " - " + msg;
	}
}

func IssuesEvent(data *gabs.Container) {
	action := data.S("action").Data().(string);

	repo, _ := data.S("repository", "full_name").Data().(string);
	user, _ := data.S("issue", "user", "login").Data().(string);
	title, _ := data.S("issue", "title").Data().(string);
	inum, _ := data.S("issue", "id").Data().(string);

	gitio := GitioShort(data.S("issue", "html_url").Data().(string));

	switch action {
		case "opened":
			message <- "[" + repo + "] " + user + " opened issue #" + inum + " \"" + title + "\" " + gitio;
		case "closed":
			message <- "[" + repo + "] " + user + " closed issue #" + inum + " \"" + title + "\" " + gitio;
		case "reopened":
			message <- "[" + repo + "] " + user + " reopened issue #" + inum + " \"" + title + "\" " + gitio;
		case "assigned":
			assignee,_ := data.S("issue", "assignee", "login").Data().(string);
			message <- "[" + repo + "] " + user + " assigned issue #" + inum + " \"" + title + "\" to " + assignee + " " + gitio;
		default:
			// Ignore it
	}
}

func PullRequestEvent(data *gabs.Container) {
	action := data.S("action").Data().(string);

	repo, _ := data.S("repository", "full_name").Data().(string);
	user, _ := data.S("pull_request", "user", "login").Data().(string);
	title, _ := data.S("pull_request", "title").Data().(string);
	inum, _ := data.S("pull_request", "number").Data().(string);

	gitio := GitioShort(data.S("pull_request", "html_url").Data().(string));

	switch action {
		case "opened":
			message <- "[" + repo + "] " + user + " opened pull request #" + inum + " \"" + title + "\" " + gitio;
		case "closed":
			if data.S("pull_request", "merged").Data().(bool) {
				message <- "[" + repo + "] " + user + " merged pull request #" + inum + " \"" + title + "\" " + gitio;
			} else {
				message <- "[" + repo + "] " + user + " closed pull request #" + inum + " \"" + title + "\" " + gitio;
			}
		case "assigned":
			assignee,_ := data.S("pull_request", "assignee", "login").Data().(string);
			message <- "[" + repo + "] " + user + " assigned pull request #" + inum + " \"" + title + "\" to " + assignee + " " + gitio;
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
		case "pull_request":
			PullRequestEvent(data);
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
	// Load config
	file, err := os.Open("gakin.json");
	if err != nil {
		fmt.Printf("[*] Unable to load config: %s\n", err.Error());
	}

	cfg, err := ioutil.ReadAll(file);
	if err != nil {
		fmt.Printf("[*] Configure error: %s\n", err.Error());
	}
	r, err := gabs.ParseJSON(cfg);
	if err != nil {
		fmt.Printf("[*] Configure error: %s\n", err.Error());
	}

	messages = make(map[string]string);

	sauce_key = r.S("sauce_key").Data().(string);

	irchndl, _ := r.S("irc").Children();
	for _, icon := range irchndl {
		server,_ := icon.S("server").Data().(string);
		channel,_ := icon.S("channel").Data().(string);
		nickname,_ := icon.S("nickname").Data().(string);

		go IRCConnection(server, channel, nickname);
	}

	http.HandleFunc("/", HandlePost);

	endpoint,_ := r.S("endpoint").Data().(string);


	http.ListenAndServe(endpoint, nil);
}

func Roll(count, sides string) (int) {
	_count,_ := strconv.Atoi(count);
	_sides,_ := strconv.Atoi(count);

	total := 0;
	for i := 0; i < _count; i++ {
		total += rand.Intn(_sides);
	}
	return total;
}

func hblookup(title string) {
	resp, err := http.Get("http://hummingbird.me/api/v1/search/anime?query="+title);
	if err != nil {
		message <- "Request Error";
	}
	defer resp.Body.Close();
	res, err := ioutil.ReadAll(resp.Body);
	if err != nil {
		message <- "Request Error";
	}
	jsn, _ := gabs.ParseJSON(res);
	ani, _ := jsn.Children();
	cnt := 1;
	for _, child := range ani {
		title :=  child.S("title").Data();
		if title == nil {
			title = "?";
		}
		status := child.S("status").Data();
		if status == nil {
			status = "?";
		}
		epcount := child.S("episode_count").Data();
		if epcount == nil {
			epcount = "?";
		}
		start := child.S("started_airing").Data();
		if start == nil {
			start = "?";
		}
		end := child.S("finished_airing").Data();
		if end == nil {
			end = "?";
		}
		slug := "https://hummingbird.me/anime/" + child.S("slug").Data().(string);
		message <- strconv.Itoa(cnt) + ") Title: " + title.(string) + " Status: " + status.(string) + " Episodes: " + strconv.FormatFloat(epcount.(float64),'f',0,64) + " Started: " + start.(string)  + " Ended: " + end.(string) + " | " + slug + "\n";
		if cnt == 5 {
			break;
		}
		cnt += 1;
	}
}

func hbuser(user string) {
	resp, err := http.Get("http://hummingbird.me/api/v1/users/"+user);
	if err != nil {
		message <- "Request Error";
	}
	defer resp.Body.Close();
	res, err := ioutil.ReadAll(resp.Body);
	if err != nil {
		message <- "Request Error";
	}
	jsn, _ := gabs.ParseJSON(res);
	name := jsn.S("name").Data();
	if name == nil {
		name = "?";
	}
	life := jsn.S("life_spent_on_anime").Data();
	if life == nil {
		life = "0";
	}
	last := jsn.S("last_library_update").Data();
	if last == nil {
		last = "?";
	}

	message <- name.(string) +": Time spent watching anime: " + strconv.FormatFloat(life.(float64) / 60 / 24 / 30,'f',2,64) + " months. Last Update: " + last.(string);
}

func sauce(imgurl string) {
	resp, err := http.Get("https://saucenao.com/search.php?db=999&output_type=2&testmode=1&numres=16&url="+imgurl+"&api_key="+sauce_key);
	if err != nil {
		message <- "Request Error";
	}
	defer resp.Body.Close();
	res, err := ioutil.ReadAll(resp.Body);
	if err != nil {
		message <- "Request Error";
	}
	jsn, _ := gabs.ParseJSON(res);
	results,_ := jsn.S("results").Children();
	cnt := 1;
	for _, child := range results {
		header := child.S("header");
		data := child.S("data");

		sim := header.S("similarity").Data();
		if sim == nil {
			sim = "?";
		}
		index_num := header.S("index_id").Data().(float64);
		index_name := header.S("index_name").Data();
		if index_name == nil {
			index_name = "?";
		}

		src_pxurl := "http://www.pixiv.net/member_illust.php?mode=medium\u0026illust_id=";

		source := "";
		artist := "?";
		title := "?";

		if index_num == 9 {
			src := data.S("source").Data();
			if src != nil {
				source = src.(string);
			}
			artists,_ := data.S("creator").Children();
			artist = artists[0].Data().(string);
		} else if index_num == 5 {
			source = src_pxurl + data.S("pixiv_id").Data().(string);
			title = data.S("title").Data().(string);
			artist = data.S("member_name").Data().(string);
		}

		message <- "[" + sim.(string) + "% Match] " + index_name.(string) + " Title: " + title + " Artist: " + artist + " Src: " + source;

		if cnt == 2 {
			break;
		}
		cnt += 1;
	}
}

func ParseCommand(conn *irc.Conn, nick, line string) {
	// Slice off the '^' and split it up
	args := strings.Split((line[1:]), " ");
	if args[0] != "" && args[0] != "^" {
		switch args[0] {
		case "ping":
			message <- nick + ", pong~";
		case "roll":
			if len(args) != 3 {
				message <- "usage: roll <num> <sides>";
				break;
			}
			message <- nick + ", " + strconv.Itoa(Roll(args[1], args[2]));
		case "hb":
			if len(args) != 3 {
				message <- "usage: hb <lookup|user> <title|username>";
			}
			if args[1] == "lookup" {
				hblookup(args[2]);
			} else if args[1] == "user" {
				hbuser(args[2]);
			} else {
				message <- "Unknown method " + args[1];
			}
		case "sauce":
			if len(args) != 2 {
				message <- "usage: sauce <image_url>";
			} else {
				sauce(args[1]);
			}
		case "notify": {
			if len(args) < 3 {
				message <- "usage: notify <nick> <message>";
			} else {
				messages[args[1]] = line[9+len(args[1]):];
				message <- "I'll let " + args[1] + " know when I see them";
			}
		}
		default:
			// Too whack to handle
		}
	}
}

func IRCConnection(host, channel, nick string) {
	IRCConnQuit := make(chan bool);
	run := true;
	cfg := irc.NewConfig(nick, nick);

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
	});

	cli.HandleFunc(irc.PRIVMSG, func(conn *irc.Conn, line *irc.Line) {
		if line.Text()[0:1] == "^" {
			ParseCommand(conn, line.Nick, line.Text());
		}
	});

	cli.HandleFunc(irc.JOIN, func(conn *irc.Conn, line *irc.Line) {
		if val, ok := messages[line.Nick]; ok {
			message <- line.Nick + ", " + val;
			delete(messages, line.Nick);
		}
	});

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
