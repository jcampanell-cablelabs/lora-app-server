package api

import (
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	pb "github.com/brocaar/lora-app-server/api"
	"github.com/brocaar/lora-app-server/internal/common"
	"github.com/brocaar/lora-app-server/internal/storage"
	"github.com/brocaar/lora-app-server/internal/test"
	"github.com/brocaar/lorawan/backend"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/net/context"
)

func TestNetworkServerAPI(t *testing.T) {
	conf := test.GetConfig()
	db, err := storage.OpenDatabase(conf.PostgresDSN)
	if err != nil {
		t.Fatal(err)
	}
	common.DB = db

	Convey("Given a clean database and api instance", t, func() {
		test.MustResetDB(common.DB)

		nsClient := test.NewNetworkServerClient()
		common.NetworkServerPool = test.NewNetworkServerPool(nsClient)

		ctx := context.Background()
		validator := &TestValidator{}
		api := NewNetworkServerAPI(validator)

		org := storage.Organization{
			Name: "test-org",
		}
		So(storage.CreateOrganization(common.DB, &org), ShouldBeNil)

		Convey("Then Create creates a network-server", func() {
			resp, err := api.Create(ctx, &pb.CreateNetworkServerRequest{
				Name:   "test ns",
				Server: "test-ns:1234",
			})
			So(err, ShouldBeNil)
			So(resp.Id, ShouldBeGreaterThan, 0)

			Convey("Then Get returns the network-server", func() {
				getResp, err := api.Get(ctx, &pb.GetNetworkServerRequest{
					Id: resp.Id,
				})
				So(err, ShouldBeNil)
				So(getResp.Name, ShouldEqual, "test ns")
				So(getResp.Server, ShouldEqual, "test-ns:1234")
			})

			Convey("Then Update updates the network-server", func() {
				_, err := api.Update(ctx, &pb.UpdateNetworkServerRequest{
					Id:     resp.Id,
					Name:   "updated-test-ns",
					Server: "updated-test-ns:1234",
				})
				So(err, ShouldBeNil)

				getResp, err := api.Get(ctx, &pb.GetNetworkServerRequest{
					Id: resp.Id,
				})
				So(err, ShouldBeNil)
				So(getResp.Name, ShouldEqual, "updated-test-ns")
				So(getResp.Server, ShouldEqual, "updated-test-ns:1234")
			})

			Convey("Then Delete deletes the network-server", func() {
				_, err := api.Delete(ctx, &pb.DeleteNetworkServerRequest{
					Id: resp.Id,
				})
				So(err, ShouldBeNil)

				_, err = api.Get(ctx, &pb.GetNetworkServerRequest{
					Id: resp.Id,
				})
				So(err, ShouldNotBeNil)
				So(grpc.Code(err), ShouldEqual, codes.NotFound)
			})

			Convey("Then List lists the network-servers", func() {
				// non admin returns nothing
				listResp, err := api.List(ctx, &pb.ListNetworkServerRequest{
					Limit:  10,
					Offset: 0,
				})
				So(err, ShouldBeNil)
				So(listResp.TotalCount, ShouldEqual, 0)
				So(listResp.Result, ShouldHaveLength, 0)

				validator.returnIsAdmin = true
				listResp, err = api.List(ctx, &pb.ListNetworkServerRequest{
					Limit:  10,
					Offset: 0,
				})
				So(err, ShouldBeNil)
				So(listResp.TotalCount, ShouldEqual, 1)
				So(listResp.Result, ShouldHaveLength, 1)
				So(listResp.Result[0].Name, ShouldEqual, "test ns")
				So(listResp.Result[0].Server, ShouldEqual, "test-ns:1234")
			})

			Convey("Given a second organization and service-profile assigned to the first organization", func() {
				org2 := storage.Organization{
					Name: "test-org-2",
				}
				So(storage.CreateOrganization(common.DB, &org2), ShouldBeNil)

				sp := storage.ServiceProfile{
					NetworkServerID: resp.Id,
					OrganizationID:  org.ID,
					Name:            "test-sp",
					ServiceProfile:  backend.ServiceProfile{},
				}
				So(storage.CreateServiceProfile(common.DB, &sp), ShouldBeNil)

				Convey("Then List with organization id lists the network-servers for the given organization id", func() {
					listResp, err := api.List(ctx, &pb.ListNetworkServerRequest{
						Limit:          10,
						OrganizationID: org.ID,
					})
					So(err, ShouldBeNil)
					So(listResp.TotalCount, ShouldEqual, 1)
					So(listResp.Result, ShouldHaveLength, 1)

					listResp, err = api.List(ctx, &pb.ListNetworkServerRequest{
						Limit:          10,
						OrganizationID: org2.ID,
					})
					So(err, ShouldBeNil)
					So(listResp.TotalCount, ShouldEqual, 0)
					So(listResp.Result, ShouldHaveLength, 0)
				})
			})
		})
	})
}
