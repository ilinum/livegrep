package server

import (
	"context"
	"fmt"

	"net"

	lngs "github.com/livegrep/livegrep/server/langserver"

	"github.com/livegrep/livegrep/server/config"
	"github.com/sourcegraph/jsonrpc2"
	"path/filepath"
	"time"
)

type ClientCapabilities struct{}

type ServerCapabilities struct{}

type InitializeParams struct {
	ProcessId        *int               `json:"processId"`
	OriginalRootPath string             `json:"originalRootPath"`
	RootPath         string             `json:"rootPath"`
	RootUri          string             `json:"rootUri"`
	Capabilities     ClientCapabilities `json:"capabilities"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
}

func GetLangServerFromFileExt(repo config.RepoConfig, filePath string) *config.LangServer {
	fileExt := filepath.Ext(filePath)
	for _, langServer := range repo.LangServers {
		for _, ext := range langServer.Extensions {
			if ext == fileExt {
				return &langServer
			}
		}
	}
	return nil
}

type LangServerClient interface {
	Initialize(params InitializeParams) (InitializeResult, error)
	JumpToDef(params *lngs.TextDocumentPositionParams) ([]lngs.Location, error)
	AllSymbols(params *lngs.DocumentSymbolParams) (result []lngs.SymbolInformation, err error)
	Hover(params *lngs.TextDocumentPositionParams) (lngs.HoverResponse, error)
}

type langServerClientImpl struct {
	rpcClient *jsonrpc2.Conn
	ctx       context.Context
}

type responseHandler struct {
}

func (r responseHandler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	// TODO
	fmt.Println("Response handler called")
}

func CreateLangServerClient(address string) (client LangServerClient, err error) {
	ctx := context.Background()
	codec := jsonrpc2.VSCodeObjectCodec{}
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return
	}
	handler := responseHandler{}
	rpcConn := jsonrpc2.NewConn(ctx, jsonrpc2.NewBufferedStream(conn, codec), handler)
	client = &langServerClientImpl{
		rpcClient: rpcConn,
		ctx:       ctx,
	}
	return client, nil
}

func (ls *langServerClientImpl) Initialize(params InitializeParams) (result InitializeResult, err error) {
	err = ls.invoke("initialize", params, &result)
	if err != nil {
		ls.invoke("initialized", nil, nil)
	}
	return
}

func (ls *langServerClientImpl) JumpToDef(params *lngs.TextDocumentPositionParams) (result []lngs.Location, err error) {
	err = ls.invoke("textDocument/definition", params, &result)
	return
}

func (ls *langServerClientImpl) AllSymbols(params *lngs.DocumentSymbolParams) (result []lngs.SymbolInformation, err error) {
	err = ls.invoke("textDocument/documentSymbol", params, &result)
	return
}

func (ls *langServerClientImpl) Hover(
	params *lngs.TextDocumentPositionParams,
) (result lngs.HoverResponse, err error) {
	err = ls.invoke("textDocument/hover", params, result)
	return
}

func (ls *langServerClientImpl) invoke(method string, params interface{}, result interface{}) error {
	start := time.Now()
	err := ls.rpcClient.Call(ls.ctx, method, params, &result)
	fmt.Printf("%s returned in %s\nResult: %+v, err: %+v\n", method, time.Since(start), result, err)
	return err
}