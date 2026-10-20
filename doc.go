// Package versioncompare 提供面向漏洞范围匹配的版本解析、比较与范围编译能力。
//
// 本包处理普通开源组件版本以及旧系统使用的通用 OS 版本。需要遵循 RPM、
// Debian、Alpine 或 Arch 原生规则时，应使用 osversion 子包；需要显式生态、
// 数字或自定义发布顺序时，应使用 versionrange 子包，旧接口不会自动切换规则。
package versioncompare
