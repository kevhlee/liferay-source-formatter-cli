package nexus

import (
	"fmt"
	"path"
	"strings"
)

const (
	PublicReposUrl = "https://repository.liferay.com/nexus/content/repositories/liferay-public-releases"
)

func GetJarFileUrl(group, artifact, version string) string {
	filename := fmt.Sprintf("%s-%s.jar", artifact, version)

	return fmt.Sprintf(
		"%s/%s",
		PublicReposUrl,
		path.Join(strings.ReplaceAll(group, ".", "/"), artifact, version, filename),
	)
}

func GetLiferayJarFileUrl(artifact, version string) string {
	return GetJarFileUrl("com.liferay", artifact, version)
}

func GetLiferayPortalJarFileUrl(artifact, version string) string {
	return GetJarFileUrl("com.liferay.portal", artifact, version)
}
