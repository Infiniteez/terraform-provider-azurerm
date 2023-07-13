package scopeconnections

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See NOTICE.txt in the project root for license information.

type ScopeConnectionProperties struct {
	ConnectionState *ScopeConnectionState `json:"connectionState,omitempty"`
	Description     *string               `json:"description,omitempty"`
	ResourceId      *string               `json:"resourceId,omitempty"`
	TenantId        *string               `json:"tenantId,omitempty"`
}
