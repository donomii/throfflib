// rpcserver.go
package throfflib

import "io"
import "os"
import "net/http"
import "net"
import "net/rpc"
import "net/rpc/jsonrpc"

import "bytes"
import "log"
import (
	"fmt"
)

func (t *TagResponder) Eval(args *Args, reply *StatusReply) error {

	code := args.A

	var en = MakeEngine()
	en = en.RunFile("bootstrap.lib")
	emit(fmt.Sprintf("code: %v\n", code))

	en = en.RunString(code, "eval")
	var ret, _ = popStack(en.dataStack)

	var rethash = map[string]string{}
	rethash["test"] = "worked"
	rethash["retval"] = ret.getSource()
	//fmt.Printf("return hash: %v\n", ret)
	for k, v := range ret._hashVal {
		rethash[k] = v.GetString()
	}

	reply.Answer = rethash
	//reply.TagsToFilesHisto, reply.TopTags = sumariseDatabase()
	emit(fmt.Sprintln("Status handler complete"))
	return nil
}

type Args struct {
	A     string
	Limit int
}

type StatusReply struct {
	Answer map[string]string
}

type TagResponder int

// NewRPCRequest returns a new rpcRequest.

// rpcRequest represents a RPC request.
// rpcRequest implements the io.ReadWriteCloser interface.
type rpcRequest struct {
	r    io.Reader     // holds the JSON formated RPC request
	rw   io.ReadWriter // holds the JSON formated RPC response
	done chan bool     // signals then end of the RPC request
}

func NewRPCRequest(r io.Reader) *rpcRequest {
	var buf bytes.Buffer
	done := make(chan bool)
	return &rpcRequest{r, &buf, done}
}

// Read implements the io.ReadWriteCloser Read method.
func (r *rpcRequest) Read(p []byte) (n int, err error) {
	return r.r.Read(p)
}

// Write implements the io.ReadWriteCloser Write method.
func (r *rpcRequest) Write(p []byte) (n int, err error) {
	r.done <- true
	return r.rw.Write(p)
}

// Close implements the io.ReadWriteCloser Close method.
func (r *rpcRequest) Close() error {
	return nil
}

// Call invokes the RPC request, waits for it to complete, and returns the results.
func (r *rpcRequest) Call() io.Reader {
	if debug {
		emit(fmt.Sprintf("Processing json rpc request\n"))
	}
	arith := new(TagResponder)

	server := rpc.NewServer()
	server.Register(arith)

	//server.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)

	go server.ServeCodec(jsonrpc.NewServerCodec(r))
	//go jsonrpc.ServeConn(r)
	<-r.done
	//b := []byte{}
	//_, _ = r.rw.Read(b)
	if debug {
		emit(fmt.Sprintln("Returning"))
	}
	return r.rw
}

func rpc_server(serverAddress string) {
	arith := new(TagResponder)

	server := rpc.NewServer()
	server.Register(arith)

	//server.HandleHTTP(rpc.DefaultRPCPath, rpc.DefaultDebugPath)

	l, e := net.Listen("tcp", serverAddress)
	if e != nil {
		log.Fatal("listen error:", e)
	}

	http.HandleFunc("/rpc", func(w http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		res := NewRPCRequest(req.Body).Call()
		io.Copy(w, res)
	})

	cwd, _ := os.Getwd()
	emit(fmt.Sprintf("Serving /files/ from:%s\n", cwd))

	http.Handle("/", http.FileServer(http.Dir("throffweb")))
	http.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir(cwd))))

	go http.ListenAndServe(":80", nil)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		//log.Println("Got connection")
		go server.ServeCodec(jsonrpc.NewServerCodec(conn))
		//log.Println("Sent response, probably")
	}

}
