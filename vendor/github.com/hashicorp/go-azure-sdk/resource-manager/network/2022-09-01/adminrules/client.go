package adminrules

import (
	"fmt"

	"github.com/hashicorp/go-azure-sdk/sdk/client/resourcemanager"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
)

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See NOTICE.txt in the project root for license information.

type AdminRulesClient struct {
	Client *resourcemanager.Client
}

func NewAdminRulesClientWithBaseURI(api environments.Api) (*AdminRulesClient, error) {
	client, err := resourcemanager.NewResourceManagerClient(api, "adminrules", defaultApiVersion)
	if err != nil {
		return nil, fmt.Errorf("instantiating AdminRulesClient: %+v", err)
	}

	return &AdminRulesClient{
		Client: client,
	}, nil
}
