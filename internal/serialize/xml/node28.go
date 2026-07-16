package xml

import "github.com/mRemoteNG/mremoteng-go/internal/connection"

func (d nodeDecoder) decode28(info *connection.ConnectionInfo, a attributes) {
	info.Raw.RedirectDiskDrives = diskDrivesValue(a, "RedirectDiskDrives")
	info.Raw.RedirectDiskDrivesCustom = a.string("RedirectDiskDrivesCustom")
	info.Inheritance.RedirectDiskDrivesCustom = a.boolean("InheritRedirectDiskDrivesCustom")
	info.Raw.EnvironmentTags = a.string("EnvironmentTags")
	info.Inheritance.EnvironmentTags = a.boolean("InheritEnvironmentTags")
}
