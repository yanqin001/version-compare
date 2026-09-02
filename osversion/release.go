package osversion

import "strings"

func splitEmbeddedMetadata(value string) (version, osVersion, architecture string) {
	version = strings.TrimSpace(value)
	if slash := strings.LastIndexByte(version, '/'); slash >= 0 {
		architecture = strings.TrimSpace(version[slash+1:])
		version = strings.TrimSpace(version[:slash])
	}

	start := 0
	for position := 0; position <= len(version); position++ {
		if position < len(version) && version[position] != '.' && version[position] != '-' && version[position] != '_' && version[position] != '+' {
			continue
		}
		if position > start {
			token := strings.ToLower(version[start:position])
			if isDistributionReleaseToken(token) {
				osVersion = token
			}
		}
		start = position + 1
	}
	return version, osVersion, architecture
}

func isDistributionReleaseToken(value string) bool {
	prefixes := [...]string{"an", "el", "fc", "oe", "amzn", "alinux", "sles", "lp"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) && allReleaseDigits(value[len(prefix):]) {
			return true
		}
	}
	return false
}

func normalizeOSRelease(distribution, value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return value
	}
	key := canonicalDistribution(normalizeDistribution(distribution))

	// 只处理发行版名称和稳定标记，不改写版本内部有排序含义的字符。
	removals := []string{" lts", " els"}
	for _, suffix := range removals {
		value = strings.TrimSpace(strings.TrimSuffix(value, suffix))
	}
	switch {
	case key == "anolis":
		value = trimReleasePrefix(value, "anolis os", "anolisos", "anolis", "an")
	case key == "ubuntu":
		value = trimReleasePrefix(value, "ubuntu")
	case key == "debian" || key == "deb" || key == "dpkg":
		value = trimReleasePrefix(value, "debian")
	case key == "rhel" || key == "centos" || key == "rocky" || key == "almalinux" || key == "oraclelinux":
		value = trimReleasePrefix(value, "red hat enterprise linux", "redhat", "rhel", "centos stream", "centos", "rocky linux", "rocky", "almalinux", "alma", "oracle linux", "oraclelinux", "el")
	case key == "fedora":
		value = trimReleasePrefix(value, "fedora", "fc")
	case key == "openeuler":
		value = trimReleasePrefix(value, "openeuler", "euleros", "oe")
	case key == "amazonlinux":
		value = trimReleasePrefix(value, "amazon linux", "amazonlinux", "amzn")
	case key == "alinux":
		value = trimReleasePrefix(value, "alibaba cloud linux", "alibabacloudlinux", "alinux")
	case key == "suse":
		value = trimReleasePrefix(value, "suse linux enterprise server", "suse linux enterprise", "opensuse tumbleweed", "opensuse leap", "opensuse", "sles", "sled", "suse", "lp")
	case key == "linuxmint":
		value = trimReleasePrefix(value, "linux mint", "linuxmint", "mint")
	case key == "kali":
		value = trimReleasePrefix(value, "kali linux", "kalilinux", "kali")
	case key == "raspbian":
		value = trimReleasePrefix(value, "raspbian")
	case key == "deepin":
		value = trimReleasePrefix(value, "deepin")
	case key == "uos":
		value = trimReleasePrefix(value, "uniontech os", "uniontechos", "uos")
	case key == "alpine":
		value = trimReleasePrefix(value, "alpine linux", "alpinelinux", "alpine", "v")
	case key == "wolfi":
		value = trimReleasePrefix(value, "wolfi os", "wolfios", "wolfi")
	case key == "chainguard":
		value = trimReleasePrefix(value, "chainguard")
	case key == "arch":
		value = trimReleasePrefix(value, "arch linux", "archlinux", "arch")
	case key == "manjaro":
		value = trimReleasePrefix(value, "manjaro linux", "manjaro")
	case key == "endeavouros":
		value = trimReleasePrefix(value, "endeavouros")
	}
	return strings.TrimSpace(value)
}

func trimReleasePrefix(value string, prefixes ...string) string {
	// 某些展示值会叠加发行版名和版本标记，例如 "Alpine Linux v3.20"；
	// 按顺序反复剥离已知前缀，直到剩余部分可直接交给原生版本解析器。
	for {
		trimmed := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(value, prefix) {
				value = strings.TrimSpace(value[len(prefix):])
				trimmed = true
				break
			}
		}
		if !trimmed {
			return value
		}
	}
}

func allReleaseDigits(value string) bool {
	if value == "" {
		return false
	}
	for index := range value {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	return true
}

func distributionMatches(candidate, constraint Distribution) bool {
	if strings.TrimSpace(string(candidate)) == "" {
		return true
	}
	constraintKey := normalizeDistribution(string(constraint))
	if constraintKey == "" || constraintKey == "os" || constraintKey == "generic" {
		return true
	}
	candidateKey := normalizeDistribution(string(candidate))
	if candidateKey == constraintKey {
		return true
	}
	constraintAlgorithm := resolveAlgorithm(string(constraint))
	canonicalConstraint := canonicalDistribution(constraintKey)
	if isAlgorithmSelector(canonicalConstraint) {
		return resolveAlgorithm(string(candidate)) == constraintAlgorithm
	}
	return canonicalDistribution(candidateKey) == canonicalConstraint
}

func isAlgorithmSelector(value string) bool {
	switch value {
	case "rpm", "deb", "dpkg", "apk", "alpm", "pacman":
		return true
	default:
		return false
	}
}

func canonicalDistribution(value string) string {
	// 兼容旧 Type 中 os:rhel、os_ubuntu 等写法；只对已知算法名剥离 os，
	// 避免把名为 oscustom 的未知发行版错误改写成另一个标识。
	if strings.HasPrefix(value, "os") && len(value) > len("os") {
		candidate := value[len("os"):]
		if _, ok := lookupAlgorithm(candidate); ok {
			value = candidate
		}
	}
	switch value {
	case "redhat", "redhatenterprise", "redhatenterpriselinux":
		return "rhel"
	case "anolisos", "openanolis":
		return "anolis"
	case "centoslinux", "centosstream":
		return "centos"
	case "rockylinux":
		return "rocky"
	case "alma":
		return "almalinux"
	case "ol":
		return "oraclelinux"
	case "amzn":
		return "amazonlinux"
	case "alibabacloudlinux":
		return "alinux"
	case "euleros":
		return "openeuler"
	case "mint":
		return "linuxmint"
	case "kalilinux":
		return "kali"
	case "uniontechos":
		return "uos"
	case "alpinelinux":
		return "alpine"
	case "archlinux":
		return "arch"
	case "opensuse", "opensuseleap", "opensusetumbleweed", "sles", "sled", "suselinuxenterprise":
		return "suse"
	default:
		return value
	}
}
