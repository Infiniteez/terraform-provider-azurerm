package web

import (
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/web/mgmt/2019-08-01/web"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"

	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/azure"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/clients"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/locks"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/services/network"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/internal/timeouts"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/utils"
)

func resourceArmAppServiceVirtualNetworkSwiftConnectionSlot() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmAppServiceVirtualNetworkSwiftConnectionSlotCreateUpdate,
		Read:   resourceArmAppServiceVirtualNetworkSwiftConnectionSlotRead,
		Update: resourceArmAppServiceVirtualNetworkSwiftConnectionSlotCreateUpdate,
		Delete: resourceArmAppServiceVirtualNetworkSwiftConnectionSlotDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Read:   schema.DefaultTimeout(5 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"app_service_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: azure.ValidateResourceID,
			},
			"subnet_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     false,
				ValidateFunc: azure.ValidateResourceID,
			},
			"slot_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceArmAppServiceVirtualNetworkSwiftConnectionSlotCreateUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Web.AppServicesClient
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Get("app_service_id").(string))
	if err != nil {
		return fmt.Errorf("Error parsing Azure Resource ID %q", id)
	}
	subnetID, err := azure.ParseAzureResourceID(d.Get("subnet_id").(string))
	if err != nil {
		return fmt.Errorf("Error parsing Azure Resource ID %q", subnetID)
	}
	resourceGroup := id.ResourceGroup
	name := id.Path["sites"]
	subnetName := subnetID.Path["subnets"]
	virtualNetworkName := subnetID.Path["virtualNetworks"]
	slotName := d.Get("slot_name").(string)

	locks.ByName(virtualNetworkName, network.VirtualNetworkResourceName)
	defer locks.UnlockByName(virtualNetworkName, network.VirtualNetworkResourceName)

	locks.ByName(subnetName, network.SubnetResourceName)
	defer locks.UnlockByName(subnetName, network.SubnetResourceName)

	appServiceExists, err := client.Get(ctx, resourceGroup, name)
	if err != nil {
		if utils.ResponseWasNotFound(appServiceExists.Response) {
			return fmt.Errorf("Error retrieving existing App Service %q (Resource Group %q): App Service not found in resource group", name, resourceGroup)
		}
		return fmt.Errorf("Error retrieving existing App Service %q (Resource Group %q): %s", name, resourceGroup, err)
	}

	slotExists, err := client.GetSlot(ctx, resourceGroup, name, slotName)
	if err != nil {
		if utils.ResponseWasNotFound(slotExists.Response) {
			return fmt.Errorf("Error retrieving existing App Service Slot %q (App Service %q / Resource Group %q): App Service not found in resource group", slotName, name, resourceGroup)
		}
		return fmt.Errorf("Error retrieving existing App Service Slot %q (App Service %q / Resource Group %q): %s", slotName, name, resourceGroup, err)
	}

	connectionEnvelope := web.SwiftVirtualNetwork{
		SwiftVirtualNetworkProperties: &web.SwiftVirtualNetworkProperties{
			SubnetResourceID: utils.String(d.Get("subnet_id").(string)),
		},
	}
	if _, err = client.CreateOrUpdateSwiftVirtualNetworkConnectionSlot(ctx, resourceGroup, name, connectionEnvelope, slotName); err != nil {
		return fmt.Errorf("Error creating/updating App Service Slot VNet association between %q (App Service %q / Resource Group %q) and Virtual Network %q: %s", slotName, name, resourceGroup, virtualNetworkName, err)
	}

	read, err := client.GetSwiftVirtualNetworkConnectionSlot(ctx, resourceGroup, name, slotName)
	if err != nil {
		return fmt.Errorf("Error retrieving App Service Slot VNet association between %q (App Service %q / Resource Group %q) and Virtual Network %q: %s", slotName, name, resourceGroup, virtualNetworkName, err)
	}
	d.SetId(*read.ID)

	return resourceArmAppServiceVirtualNetworkSwiftConnectionSlotRead(d, meta)
}

func resourceArmAppServiceVirtualNetworkSwiftConnectionSlotRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Web.AppServicesClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Id())
	if err != nil {
		return fmt.Errorf("Error parsing Azure Resource ID %q", id)
	}
	resourceGroup := id.ResourceGroup
	name := id.Path["sites"]
	slotName := id.Path["slots"]

	slot, err := client.GetSlot(ctx, resourceGroup, name, slotName)
	if err != nil {
		if utils.ResponseWasNotFound(slot.Response) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error retrieving existing App Service Slot %q (App Service %q / Resource Group %q): %s", slotName, name, resourceGroup, err)
	}
	appService, err := client.Get(ctx, resourceGroup, name)
	if err != nil {
		if utils.ResponseWasNotFound(appService.Response) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error retrieving existing App Service %q (Resource Group %q): %s", name, resourceGroup, err)
	}
	swiftVnet, err := client.GetSwiftVirtualNetworkConnectionSlot(ctx, resourceGroup, name, slotName)
	if err != nil {
		if utils.ResponseWasNotFound(swiftVnet.Response) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error retrieving App Service Slot VNet association for %q (App Service %q / Resource Group %q): %s", slotName, name, resourceGroup, err)
	}

	if swiftVnet.SwiftVirtualNetworkProperties == nil {
		return fmt.Errorf("Error retrieving virtual network properties (Slot Name %q / App Service %q / Resource Group %q): `properties` was nil", slotName, name, resourceGroup)
	}
	props := *swiftVnet.SwiftVirtualNetworkProperties
	subnetID := props.SubnetResourceID
	if subnetID == nil || *subnetID == "" {
		d.SetId("")
		return nil
	}
	d.Set("subnet_id", subnetID)
	d.Set("app_service_id", appService.ID)
	d.Set("slot_name", slot.Name)
	return nil
}

func resourceArmAppServiceVirtualNetworkSwiftConnectionSlotDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).Web.AppServicesClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := azure.ParseAzureResourceID(d.Get("app_service_id").(string))
	if err != nil {
		return fmt.Errorf("Error parsing Azure Resource ID %q", id)
	}
	subnetID, err := azure.ParseAzureResourceID(d.Get("subnet_id").(string))
	if err != nil {
		return fmt.Errorf("Error parsing Azure Resource ID %q", subnetID)
	}
	slotName := d.Get("slot_name").(string)
	resourceGroup := id.ResourceGroup
	name := id.Path["sites"]
	subnetName := subnetID.Path["subnets"]
	virtualNetworkName := subnetID.Path["virtualNetworks"]

	locks.ByName(virtualNetworkName, network.VirtualNetworkResourceName)
	defer locks.UnlockByName(virtualNetworkName, network.VirtualNetworkResourceName)

	locks.ByName(subnetName, network.SubnetResourceName)
	defer locks.UnlockByName(subnetName, network.SubnetResourceName)

	read, err := client.GetSwiftVirtualNetworkConnectionSlot(ctx, resourceGroup, name, slotName)
	if err != nil {
		return fmt.Errorf("Error making read request on virtual network properties (Slot Name %q / App Service %q / Resource Group %q): %+v", slotName, name, resourceGroup, err)
	}
	if read.SwiftVirtualNetworkProperties == nil {
		return fmt.Errorf("Error retrieving virtual network properties (Slot Name %q / App Service %q / Resource Group %q): `properties` was nil", slotName, name, resourceGroup)
	}
	props := *read.SwiftVirtualNetworkProperties
	subnet := props.SubnetResourceID
	if subnet == nil || *subnet == "" {
		// assume deleted
		return nil
	}

	resp, err := client.DeleteSwiftVirtualNetworkSlot(ctx, resourceGroup, name, slotName)
	if err != nil {
		if !utils.ResponseWasNotFound(resp) {
			return fmt.Errorf("Error deleting virtual network properties (Slot Name %q / App Service %q / Resource Group %q): %+v", slotName, name, resourceGroup, err)
		}
	}

	return nil
}
