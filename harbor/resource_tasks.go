package harbor

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/nolte/terraform-provider-harbor/gen/harborctl/client"
	"github.com/nolte/terraform-provider-harbor/gen/harborctl/client/products"
	"github.com/nolte/terraform-provider-harbor/gen/harborctl/models"
)

func resourceTasks() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"vulnerability_scan_policy": {
				Type:     schema.TypeString,
				Required: true,
			},
			"cron_schedule": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
		Create: resourceTasksCreate,
		Read:   resourceTasksRead,
		Update: resourceTasksUpdate,
		Delete: resourceTasksDelete,
	}
}

func resourceTasksCreate(d *schema.ResourceData, m interface{}) error {
	scanScheduleType, _, err := readSystemScanAllSchedule(m)
	if err != nil {
		return err
	}

	if scanScheduleType == "None" {
		err = createSystemScanAllSchedule(d, m)
	} else {
		err = updateSystemScanAllSchedule(d, m)
	}
	if err != nil {
		return err
	}

	d.SetId(resource.PrefixedUniqueId(fmt.Sprintf("%s-", "vulnerability_scan")))

	return resourceTasksRead(d, m)
}

func resourceTasksRead(d *schema.ResourceData, m interface{}) error {
	scanScheduleType, scanScheduleCron, err := readSystemScanAllSchedule(m)
	if err != nil {
		return err
	}

	if err := d.Set("vulnerability_scan_policy", strings.ToLower(scanScheduleType)); err != nil {
		return err
	}

	if scanScheduleType == "Custom" {
		if err := d.Set("cron_schedule", scanScheduleCron); err != nil {
			return err
		}
	}

	return nil
}

func resourceTasksUpdate(d *schema.ResourceData, m interface{}) error {
	scanScheduleType, _, err := readSystemScanAllSchedule(m)
	if err != nil {
		return err
	}

	if scanScheduleType == "None" {
		err = createSystemScanAllSchedule(d, m)
	} else {
		err = updateSystemScanAllSchedule(d, m)
	}
	if err != nil {
		return err
	}

	return resourceTasksRead(d, m)
}

func resourceTasksDelete(d *schema.ResourceData, m interface{}) error {
	apiClient := m.(*client.Harbor)

	scanScheduleType, _, err := readSystemScanAllSchedule(m)
	if err != nil {
		return err
	}

	if scanScheduleType == "None" {
		return nil
	}
	// There is no API for deleting the system scanning policy. Instead, we update it to 'None'.
	body := &models.AdminJobSchedule{
		Schedule: &models.AdminJobScheduleObj{
			Cron: "",
			Type: "None",
		},
	}

	params := products.NewPutSystemScanAllScheduleParams().WithSchedule(body)

	_, err = apiClient.Products.PutSystemScanAllSchedule(params, nil)
	if err != nil {
		return err
	}

	return nil
}

func readSystemScanAllSchedule(m interface{}) (string, string, error) {
	apiClient := m.(*client.Harbor)

	resp, err := apiClient.Products.GetSystemScanAllSchedule(products.NewGetSystemScanAllScheduleParams(), nil)
	if err != nil {
		return "", "", err
	}

	if resp.Payload.Schedule == nil {
		return "None", "", nil
	}

	return resp.Payload.Schedule.Type, resp.Payload.Schedule.Cron, nil
}

func createSystemScanAllSchedule(d *schema.ResourceData, m interface{}) error {
	apiClient := m.(*client.Harbor)

	schedule, err := getSchedule(d)
	if err != nil {
		return err
	}

	body := &models.AdminJobSchedule{
		Schedule: schedule,
	}

	params := products.NewPostSystemScanAllScheduleParams().WithSchedule(body)

	_, err = apiClient.Products.PostSystemScanAllSchedule(params, nil)
	if err != nil {
		return err
	}

	return nil
}

func updateSystemScanAllSchedule(d *schema.ResourceData, m interface{}) error {
	apiClient := m.(*client.Harbor)

	schedule, err := getSchedule(d)
	if err != nil {
		return err
	}

	body := &models.AdminJobSchedule{
		Schedule: schedule,
	}

	params := products.NewPutSystemScanAllScheduleParams().WithSchedule(body)

	_, err = apiClient.Products.PutSystemScanAllSchedule(params, nil)
	if err != nil {
		return err
	}

	return nil
}

func getSchedule(d *schema.ResourceData) (*models.AdminJobScheduleObj, error) {
	scanPolicy := d.Get("vulnerability_scan_policy").(string)

	switch scanPolicy {
	case "hourly":
		return &models.AdminJobScheduleObj{
			Cron: "0 0 * * * *",
			Type: "Hourly",
		}, nil
	case "daily":
		return &models.AdminJobScheduleObj{
			Cron: "0 0 0 * * *",
			Type: "Daily",
		}, nil
	case "weekly":
		return &models.AdminJobScheduleObj{
			Cron: "0 0 0 * * 0",
			Type: "Weekly",
		}, nil
	case "custom":
		return &models.AdminJobScheduleObj{
			Cron: d.Get("cron_schedule").(string),
			Type: "Custom",
		}, nil
	}

	errMsg := fmt.Sprintf("Invalid scan policy: %s. Valid values are: hourly, daily, weekly, custom.", scanPolicy)
	return &models.AdminJobScheduleObj{}, errors.New(errMsg)
}
