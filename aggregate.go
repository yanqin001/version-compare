package versioncompare

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/yanqin001/version-compare/internal/versioncore"
)

type parsedItem struct {
	original string
	version  versioncore.Version
}

// GetMaxVersion 返回 versions 中最大的原始版本字符串。
// 空列表、无效版本或不可比较版本会打印日志并返回错误。
func (svc *versionSvc) GetMaxVersion(versions []string) (string, error) {
	items, err := parseItems(versions)
	if err != nil {
		return "", logFailure("GetMaxVersion", err)
	}
	maximum := items[0]
	for _, item := range items[1:] {
		result, compareErr := versioncore.Compare(maximum.version, item.version, false)
		if compareErr != nil {
			return "", logFailure("GetMaxVersion", translateVersionError(compareErr))
		}
		if result < 0 {
			maximum = item
		}
	}
	return maximum.original, nil
}

// GetMinVersion 返回 versions 中最小的原始版本字符串。
// 等价版本保留最先出现的字符串；空列表、无效版本或不可比较版本会打印日志并返回错误。
func (svc *versionSvc) GetMinVersion(versions []string) (string, error) {
	items, err := parseItems(versions)
	if err != nil {
		return "", logFailure("GetMinVersion", err)
	}
	minimum := items[0]
	for _, item := range items[1:] {
		result, compareErr := versioncore.Compare(minimum.version, item.version, false)
		if compareErr != nil {
			return "", logFailure("GetMinVersion", translateVersionError(compareErr))
		}
		if result > 0 {
			minimum = item
		}
	}
	return minimum.original, nil
}

// Sort 对 versions 进行稳定升序原地排序，并返回同一个切片。
// 空切片直接成功返回。实现会先完成全部解析；版本无效或不可比较时返回错误，
// 且不会部分修改输入切片。
func (svc *versionSvc) Sort(versions []string) ([]string, error) {
	if len(versions) == 0 {
		return versions, nil
	}
	items, err := parseItems(versions)
	if err != nil {
		return nil, logFailure("Sort", err)
	}
	if err = validateComparable(items); err != nil {
		return nil, logFailure("Sort", err)
	}
	sort.SliceStable(items, func(left, right int) bool {
		result, _ := versioncore.Compare(items[left].version, items[right].version, false)
		return result < 0
	})
	for index := range items {
		versions[index] = items[index].original
	}
	return versions, nil
}

// GetClosestVersion 返回 versions 中与 version 加权数值距离最近的版本。
// 目标位于列表范围外时返回对应边界，距离相同时返回较高版本；空列表、无效版本
// 或不可比较版本会打印日志并返回错误。函数不会修改 versions。
func (svc *versionSvc) GetClosestVersion(version string, versions []string) (string, error) {
	if len(versions) == 0 {
		return "", logFailure("GetClosestVersion", ErrEmptyVersionList)
	}
	target, err := versioncore.Parse(version, versioncore.Normal)
	if err != nil {
		return "", logFailure("GetClosestVersion", translateVersionError(err))
	}
	items, err := parseItems(versions)
	if err != nil {
		return "", logFailure("GetClosestVersion", err)
	}

	for _, item := range items {
		result, compareErr := versioncore.Compare(target, item.version, false)
		if compareErr == nil && result == 0 {
			return item.original, nil
		}
	}
	if target.IsOpaque() {
		return "", logFailure("GetClosestVersion", fmt.Errorf("%w: %q", ErrIncomparableVersion, version))
	}
	if err = validateComparable(items); err != nil {
		return "", logFailure("GetClosestVersion", err)
	}
	sort.SliceStable(items, func(left, right int) bool {
		result, _ := versioncore.Compare(items[left].version, items[right].version, false)
		return result < 0
	})

	insertAt := sort.Search(len(items), func(index int) bool {
		result, _ := versioncore.Compare(items[index].version, target, false)
		return result >= 0
	})
	if insertAt == 0 {
		return items[0].original, nil
	}
	if insertAt == len(items) {
		return items[len(items)-1].original, nil
	}

	leftDistance := versionDistance(target, items[insertAt-1].version)
	rightDistance := versionDistance(target, items[insertAt].version)
	if rightDistance.Cmp(leftDistance) <= 0 {
		return items[insertAt].original, nil
	}
	return items[insertAt-1].original, nil
}

func parseItems(versions []string) ([]parsedItem, error) {
	if len(versions) == 0 {
		return nil, ErrEmptyVersionList
	}
	items := make([]parsedItem, len(versions))
	for index, value := range versions {
		parsed, err := versioncore.Parse(value, versioncore.Normal)
		if err != nil {
			return nil, translateVersionError(err)
		}
		items[index] = parsedItem{original: value, version: parsed}
	}
	return items, nil
}

func validateComparable(items []parsedItem) error {
	for index := 1; index < len(items); index++ {
		if _, err := versioncore.Compare(items[0].version, items[index].version, false); err != nil {
			return translateVersionError(err)
		}
	}
	return nil
}

func versionDistance(left, right versioncore.Version) *big.Int {
	leftParts := left.MainSegments()
	rightParts := right.MainSegments()
	length := len(leftParts)
	if len(rightParts) > length {
		length = len(rightParts)
	}

	distance := new(big.Int)
	weight := big.NewInt(1)
	base := big.NewInt(1000)
	for index := length - 1; index >= 0; index-- {
		leftValue := new(big.Int)
		rightValue := new(big.Int)
		if index < len(leftParts) {
			leftValue.SetString(leftParts[index], 10)
		}
		if index < len(rightParts) {
			rightValue.SetString(rightParts[index], 10)
		}
		difference := new(big.Int).Sub(leftValue, rightValue)
		difference.Abs(difference)
		difference.Mul(difference, weight)
		distance.Add(distance, difference)
		weight.Mul(weight, base)
	}
	return distance
}
