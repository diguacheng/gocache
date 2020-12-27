package gocache

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"gocache/consistenthash"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	pb "gocache/gocachepb"
)


const (
	defaultBasePath = "/_gocache/"
	defaultReplicas=50
)

type HTTPPool struct {
	//this peer's base URL, e.g. "https://example.net:8000"
	self     string // 用来记录自己的地址
	basePath string // 用来记录节点间通信地址的前缀
	mu sync.Mutex // 用于守护
	peers *consistenthash.Map // 一致性算法的map 
	httpGetter map[string]*httpGetter //   keyed by e.g. "http://10.0.0.2:8008"
}

func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s",p.self,fmt.Sprintf(format,v...))
}

func (p *HTTPPool)ServeHTTP(w http.ResponseWriter,r *http.Request){
	if !strings.HasPrefix(r.URL.Path,p.basePath){
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	p.Log("%s %s", r.Method, r.URL.Path)
	// /<basepath>/<groupname>/<key> required 
	
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}
	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	body,err:=proto.Marshal(&pb.Response{Value: view.ByteSlice()})
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(body)
}

// 更新httppool的节点列表 
func (p *HTTPPool)Set(peers ...string){
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers=consistenthash.New(defaultReplicas,nil)
	p.peers.Add(peers...)
	p.httpGetter=make(map[string]*httpGetter,len(peers)) 
	for _,peer :=range peers{
		// todo 这里有疑问
		p.httpGetter[peer]=&httpGetter{baseURL: peer+p.basePath}
	}

}

// 选择远处的节点 不能为空 也不能为节点自己
func (p *HTTPPool)PickPeer(key string)(PeerGetter,bool){
	p.mu.Lock()
	defer p.mu.Unlock() 
	if peer:=p.peers.Get(key);peer!=""&&peer!=p.self{
		p.Log("Pick peer %s",peer)
		return p.httpGetter[peer],true
	}
	return nil,false
}

var _ PeerPicker=(*HTTPPool)(nil)




type httpGetter struct {
	baseURL string
}

func (h *httpGetter)Get(in *pb.Request,out *pb.Response)error{
	u:=fmt.Sprintf("%v%v/%v",
		h.baseURL,url.QueryEscape(in.Group),url.QueryEscape(in.Key))
	res,err:=http.Get(u)
	if err!=nil{
		return err
	}
	defer res.Body.Close()

	if res.StatusCode!=http.StatusOK{
		return fmt.Errorf("server returned:%v",res.StatusCode)
	}

	bytes,err:=ioutil.ReadAll(res.Body)
	if err=proto.Unmarshal(bytes,out);err!=nil{
		return fmt.Errorf("decoding response body: %v", err)
	}
	return nil
}

var _ PeerGetter=(*httpGetter)(nil)