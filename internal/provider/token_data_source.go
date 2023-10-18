package provider

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jcmturner/gokrb5/v8/client"
	"github.com/jcmturner/gokrb5/v8/config"
	"github.com/jcmturner/gokrb5/v8/spnego"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &TokenDataSource{}

func NewTokenDataSource() datasource.DataSource {
	return &TokenDataSource{}
}

// TokenDataSource defines the data source implementation.
type TokenDataSource struct {
}

// TokenDataSourceModel describes the data source data model.
type TokenDataSourceModel struct {
	Username               types.String `tfsdk:"username"`
	Password               types.String `tfsdk:"password"`
	Realm                  types.String `tfsdk:"realm"`
	Service                types.String `tfsdk:"service"`
	Kdc                    types.String `tfsdk:"kdc"`
	DisableFASTNegotiation types.Bool   `tfsdk:"disable_fast_negotiation"`
	Token                  types.String `tfsdk:"token"`
	Id                     types.String `tfsdk:"id"`
}

func (r *TokenDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_token"
}

func (r *TokenDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Retrieves a Kerberos SPNEGO token.",

		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				MarkdownDescription: "The username to authenticate to Kerberos with.",
				Required:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "The password to authenticate to Kerberos with.",
				Required:            true,
				Sensitive:           true,
			},
			"realm": schema.StringAttribute{
				MarkdownDescription: "The realm to which the Kerberos principal belongs.",
				Required:            true,
			},
			"service": schema.StringAttribute{
				MarkdownDescription: "The name of the Kerberos service to authenticate to.",
				Required:            true,
			},
			"kdc": schema.StringAttribute{
				MarkdownDescription: "The address of the KDC to authenticate to.",
				Required:            true,
			},
			"disable_fast_negotiation": schema.BoolAttribute{
				MarkdownDescription: "Whether to disable FAST pre-authentication negotiation.",
				Optional:            true,
				Computed:            true,
			},
			"token": schema.StringAttribute{
				MarkdownDescription: "The Kerberos SPNEGO token.",
				Computed:            true,
				Sensitive:           true,
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of the Kerberos token.",
				Computed:            true,
			},
		},
	}
}

func (r *TokenDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
}

func (r *TokenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data *TokenDataSourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	settings := []func(*client.Settings){
		client.AssumePreAuthentication(true),
	}

	if data.DisableFASTNegotiation.ValueBool() {
		settings = append(settings, client.DisablePAFXFAST(true))
	}

	krb5Conf := config.New()
	krb5Conf.Realms = append(krb5Conf.Realms, config.Realm{
		Realm: data.Realm.ValueString(),
		KDC:   []string{data.Kdc.ValueString()},
	})

	cl := client.NewWithPassword(data.Username.ValueString(), data.Realm.ValueString(), data.Password.ValueString(), krb5Conf, settings...)

	if err := cl.Login(); err != nil {
		resp.Diagnostics.AddError("Unable to log in", err.Error()) // TODO: better error messaging
		return
	}

	defer cl.Destroy()

	spnegoClient := spnego.SPNEGOClient(cl, data.Service.ValueString())

	if err := spnegoClient.AcquireCred(); err != nil {
		resp.Diagnostics.AddError("Unable to acquire SPNEGO credential", err.Error()) // TODO: better error messaging
		return
	}

	spnegoToken, err := spnegoClient.InitSecContext()

	if err != nil {
		resp.Diagnostics.AddError("Unable to initialize SPNEGO context", err.Error()) // TODO: better error messaging
		return
	}

	marshalledToken, err := spnegoToken.Marshal()
	if err != nil {
		resp.Diagnostics.AddError("Unable to marshal SPNEGO token", err.Error()) // TODO: better error messaging
		return
	}

	data.Token = types.StringValue(base64.StdEncoding.EncodeToString(marshalledToken))

	data.Id = types.StringValue(fmt.Sprintf("%s@%s", data.Username.ValueString(), data.Realm.ValueString()))

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "read a data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
