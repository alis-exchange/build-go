package build_test

import (
	context "context"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/alis-exchange/build-go/alis/build"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func authCtx(ctx context.Context) context.Context {
	client := testAuthClient()
	tokens := &build.Tokens{
		AccessToken:  os.Getenv("ALIS_ACCESS_TOKEN"),
		RefreshToken: os.Getenv("ALIS_REFRESH_TOKEN"),
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		panic("Set ALIS_ACCESS_TOKEN and ALIS_REFRESH_TOKEN environment variables")
	}
	authResp, err := client.Authenticate(tokens, time.Now())
	if err != nil {
		panic(err)
	}
	if authResp.Refreshed {
		println("Refreshed access token. Normally you would save the new access token and refresh token to your database/cache.")
	}
	return metadata.AppendToOutgoingContext(ctx, "authorization", tokens.AccessToken)
}

func Test_ListBuildSpecs(t *testing.T) {
	client, err := build.NewClient()
	if err != nil {
		t.Fatal(err)
	}
	listResp, err := client.ListBuildSpecs(authCtx(t.Context()), &build.ListBuildSpecsRequest{
		View: build.BuildSpecView_BUILD_SPEC_VIEW_FULL,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, spec := range listResp.BuildSpecs {
		println(spec.DisplayName)
	}
}

func Test_RetrieveMyWorkstation(t *testing.T) {
	client, err := build.NewClient()
	if err != nil {
		t.Fatal(err)
	}
	authCtx := authCtx(t.Context())
	op, err := client.RetrieveMyWorkstation(authCtx, &build.RetrieveMyWorkstationRequest{
		CountryCode: "ZA",
	})
	if err != nil {
		t.Fatal(err)
	}
	for {
		op, err = client.GetWorkstationOperation(authCtx, &longrunningpb.GetOperationRequest{
			Name: op.Name,
		})
		if err != nil {
			t.Fatal(err)
		}
		if op.Done {
			break
		}
		time.Sleep(5 * time.Second)
	}
	if op.GetError() != nil {
		t.Fatal(op.GetError())
	}
	resp := &build.RetrieveMyWorkstationResponse{}
	if err := anypb.UnmarshalTo(op.GetResponse(), resp, proto.UnmarshalOptions{}); err != nil {
		t.Fatal(err)
	}
	println("workstation uri", resp.GetUri())
}
