# version-compare

`version-compare` 是一个 Go 版本比较与漏洞范围匹配库。它保留旧项目使用的 `VersionSvc` 契约和可复用、并发安全的编译 Matcher；`osversion` 子包按发行版原生规则处理操作系统软件包，`versionrange` 子包提供显式分类的发布版本比较。

主要能力：

- 比较、排序以及查找普通组件版本的最小值、最大值和最近值；
- 匹配单个区间、比较符表达式、精确版本列表和多个范围的并集；
- 支持 OpenSSL `1.1.1a` 一类字母补丁号，无需单独的 OpenSSL 类型；
- 使用分支修复版本收紧漏洞上界；
- 按 RPM、Debian/dpkg、Alpine/apk 和 Arch Linux/ALPM 规则比较 OS 软件包版本；
- 联合匹配发行版、OS 版本、软件包版本和架构；
- 对未知发行版保留旧系统的 generic OS 比较行为。
- 按数字、日历、生态版本或明确的自定义顺序比较结构化范围，支持声明标签前后缀。

## 要求与安装

- Go 1.23.4 或更高版本；
- 根包和 `osversion` 只使用 Go 标准库；`versionrange` 的生态规则依赖固定版本的 `deps.dev/util/semver`。

```bash
go get github.com/yanqin001/version-compare@latest
```

根包名为 `versioncompare`：

```go
import versioncompare "github.com/yanqin001/version-compare"
```

## 快速开始

### `VersionSvc` 兼容接口

`VersionSvc` 保留原有六个方法，适合低频调用或旧项目直接迁移：

```go
svc := versioncompare.NewVersionSvc()

affected, err := svc.InExp(versioncompare.VersionExp{
    Version:    "1.1.1a",
    Expression: "[1.1.1,1.1.1t)",
})
if err != nil {
    // 库已经使用标准日志记录错误，调用方仍必须处理返回值。
    return err
}
fmt.Println(affected) // true
```

`VersionExp.Type` 支持：

- 空字符串、`normal`、`component`：普通组件规则；
- `openssl`：仅作为迁移别名，仍使用普通组件规则；
- `os`：兼容旧数据的 generic OS 规则。

需要 RPM、Debian、APK 或 ALPM 原生语义时，应使用 [`osversion`](#操作系统软件包匹配)，不要将发行版名称塞入 `VersionExp.Type`。

`Sort` 会稳定地原地修改传入切片并返回同一个切片。实现会先解析全部版本，解析失败时不会留下部分排序结果。

### 编译并复用 Matcher

对同一漏洞范围匹配大量版本时，应只编译一次：

```go
matcher, err := versioncompare.Compile(
    ">=1.1.1, <1.1.1t",
    versioncompare.CompileOptions{},
)
if err != nil {
    return err
}

for _, version := range []string{"1.1.1a", "1.1.1t", "3.0.0"} {
    affected, err := matcher.Match(version)
    if err != nil {
        return err
    }
    fmt.Printf("%s: %t\n", version, affected)
}
```

`Matcher` 编译后不可变，可由多个 goroutine 并发复用。它会复制 `CompileOptions` 中的分支修复数据，不保留调用方切片。

## 显式分类比较（v1.1.0）

新数据已声明版本体系时，使用独立子包；旧 `VersionSvc`、`Compile` 和 `osversion` API 及默认行为保持不变。

```go
import "github.com/yanqin001/version-compare/versionrange"

classification := versionrange.Classification{
    RangeType:     versionrange.TypeVersion,
    VersionScheme: "pep440",
}
comparator, err := versionrange.NewComparator(classification)
if err != nil {
    return err
}
order, err := comparator.Compare("3.15.0a1", "3.15.0rc3")
if err != nil {
    return err
}
fmt.Println(order) // -1
err = classification.ValidateRange("3.15.0a1", "3.15.0rc3", false)
```

`Compare` 返回标准 `-1/0/1`，与旧 `VersionSvc.Compare` 的 `0/1/2` 不同。`ValidateRange(begin, end, endInclusive)` 校验顺序，默认半开区间要求 `begin < end`，闭区间允许相等；实际命中判断、区间并集和业务处置由调用方负责。结构化边界直接比较，不经过表达式解析，因此固定包装中的括号不再成为表达式语法错误。

| `range_type` / `version_scheme` | 比较语义 |
| --- | --- |
| `VERSION` / `normal` | 复用根包通用解析，数字段、字母补丁和完整后缀参与比较；后缀数字按数值排序，不设 alpha/beta/rc 白名单 |
| 历史 `numeric/release/semver/calver/go/npm/cargo/nuget/composer` | `Canonical()` 统一返回 `normal`，比较行为与 normal 相同 |
| `VERSION` / `pep440`、`maven`、`rubygems` | 继续采用 `deps.dev/util/semver` 的对应生态规则 |
| `VERSION` / `dpkg`、`rpm`、`apk`、`alpm` | 复用 `osversion` 的发行版原生顺序 |
| `VERSION` / `custom` | 按 `version_order` 的完整原始标签顺序比较；清单外版本不可比较 |
| `GIT`、`UNKNOWN` | 不推断顺序，创建比较器返回 `ErrUnordered`；不查询 Git 祖先关系 |

分类字段使用同名 JSON：`range_type`、`version_scheme`、`ref_kind`、`version_prefix`、`version_suffix`、`version_order`。`VERSION` 的 `ref_kind` 只能为空或 `tag`；`GIT` 必须声明 `commit/tag/branch/alias`，且与 `UNKNOWN` 一样不携带 `version_scheme` 或数字包装。

`normal` 的 `version_prefix` / `version_suffix` 是可选规范化提示：匹配时剥离，否则继续通用解析，不再因 native 缺少上游 `v` 前缀而失败。例如 `normal + version_prefix=b` 可以比较 `b999 < b1000`；`version_suffix=(jdk8/11)` 可以剥离已声明的运行时包装。其他专用生态和 OS 算法继续要求包装匹配；`custom` 使用完整原始标签，不能配置包装。

normal 支持 `v2.10s`、`v2.13-pre4`、`v2.27-rc10`、任意可解析后缀，完整保留字母补丁和后缀；支持 `release-` / `release_`、`r1v60` / `r1rv78` 以及数字版本的 `-stable/-final/-ga/-custom` 正式版别名。40/64 位完整提交仍不可比较，命名版本使用 custom。

**v1.1.1 的排序语义变更：** 上述九个历史分类名称不再使用独立生态排序，而使用旧式通用业务规则，例如 `rc2 < rc10`，`+build.1 < +build.2`。normal 始终完整比较后缀；根包的 `CompileOptions.MatchMain` 和 `StrictBoundaries` 行为不变，项目门禁仍可显式忽略预发布后缀。需要 Python、Maven、RubyGems 原生规则时继续声明对应专用策略。

`NewComparator` 复制分类中的顺序切片，比较器可并发复用；空分类只能用于识别旧记录，创建比较器必须显式声明类型。错误可通过 `errors.Is` 判断：

- `versionrange.ErrInvalidClassification`：分类缺失、字段冲突或未知枚举。
- `versionrange.ErrUnordered`：缺少顺序、清单外标签、包装不符或 Git 引用；与根包 `ErrIncomparableVersion` 为同一错误。
- 根包 `ErrInvalidVersion`：版本不符合指定算法或使用通配符。
- 根包 `ErrInvalidExpression`：可排序区间逆序或半开空区间。

`Classification.ValidateRange` 允许 `GIT/UNKNOWN` 作为未决元数据保留，不代表这些范围已能匹配；调用方应显式处理 `ErrUnordered`，不能将其当作明确未命中。

### 依赖约束

生态规则固定使用 `deps.dev/util/semver v0.0.0-20261013100046-7bac2db56a63`，以复用已有生态实现并避免多套算法漂移；该依赖要求 Go 1.23.4。此子包不发起网络请求，不引入包管理器运行时；依赖升级必须执行生态边界及兼容测试。比较成本可用 `go test ./versionrange -run '^$' -bench . -benchmem` 复核，不使用无界全局缓存。

## 表达式语法

根包和 `osversion` 使用相同的范围语法：

| 形式 | 示例 | 含义 |
| --- | --- | --- |
| 通配 | `*` | 任意非空、可识别版本 |
| 裸精确值 | `1.2.3` | 等于 `1.2.3` |
| 方括号精确值 | `[1.2.3]` | 等于 `1.2.3` |
| 等号 | `=1.2.3`、`==1.2.3` | 等于 `1.2.3` |
| 闭/开区间 | `[1.0,2.0)` | `>=1.0 && <2.0` |
| 无界区间 | `(,2.0]`、`[1.0,)` | 只限制一侧 |
| 单比较符 | `<2.0`、`>=1.0` | 单侧范围 |
| 双条件 | `>=1.0, <2.0`、`>=1.0 <2.0` | 两个边界的交集 |
| 显式并集 | `1.0 || [2.0,3.0)` | 任一分支命中即可 |
| 相邻区间 | `(,1.0][2.0,3.0)` | 两个区间的并集 |
| 精确值列表 | `1.0,2.0,v3` | 三个精确版本的并集 |

空表达式、双空端点 `(,)`、边界逆序、缺失括号和相互冲突的比较符都会在 `Compile` 阶段返回 `ErrInvalidExpression`。

表达式中的版本边界不能包含空白、逗号或 `[]()<>=|*` 等语法保留字符。主流 RPM、Debian、APK 和 ALPM 包版本通常不需要这些字符；若只需比较包含保留字符的特殊 ALPM `pkgver`，可直接使用 `osversion.Service.Compare`。

普通组件范围的无后缀端点默认只约束主版本和字母补丁号，以兼容既有漏洞数据。例如 `[21.0.0,21.0.0]` 可以命中 `21.0.0-pre1`。显式预发布窗口 `[6.7-rc1,6.7)` 会严格比较后缀，因此不会错误命中正式版 `6.7`。

需要让无后缀端点也完整比较预发布和附加修订字段时，可启用严格边界模式：

```go
matcher, err := versioncompare.Compile("[21.0.0]", versioncompare.CompileOptions{
    StrictBoundaries: true,
})
if err != nil {
    return err
}

affected, err := matcher.Match("21.0.0-pre1")
if err != nil {
    return err
}
fmt.Println(affected) // false
```

`StrictBoundaries` 会同时作用于精确值、区间上下界和 `FixedVersions` 生成的排他上界。它只关闭自动 `MatchMain`；如果同时显式设置 `MatchMain: true`，仍按调用方的显式要求忽略预发布和附加修订字段。该选项默认关闭，因此现有调用方行为不变。

## 分支修复版本 `Fixed`

分支修复版本表示某个维护分支的首个安全版本。命中分支后，它会作为排他上界收紧漏洞范围；修复版本本身不受影响。修复版本只允许缩小原范围，不会扩大影响面。

新 API 使用明确的结构：

```go
matcher, err := versioncompare.Compile("[3.13-rc1,4.3-rc1)", versioncompare.CompileOptions{
    FixedVersions: []versioncompare.FixedVersion{
        {Branch: "3.16", Version: "3.16.35"},
        {Branch: "3.18", Version: "3.18.54"},
    },
})
```

兼容接口继续接受旧字符串：

```go
versioncompare.VersionExp{
    Version:    "4.14.271",
    Expression: "[4.0,5.0)",
    Fixed:      []string{"4.14:4.14.272"},
}
```

也可传入 `4.14.272` 并让兼容层推导 `4.14` 分支。推荐新代码始终显式填写 `Branch`。重复分支、空字段、不透明版本，以及不属于所声明分支的修复版本都会在编译时失败；存在多个可匹配分支时，优先采用更具体的分支。

## 比较、排序与聚合

`VersionSvc.Compare(v1, v2)` 成功时只返回以下三个值：

| 常量 | 整数值 | 含义 |
| --- | ---: | --- |
| `CompareLess` | `0` | `v1 < v2` |
| `CompareEqual` | `1` | `v1 == v2` |
| `CompareGreater` | `2` | `v1 > v2` |

接口返回类型为 `int` 以兼容旧调用方；与常量比较时可写成 `result == int(versioncompare.CompareLess)`。无法形成可靠顺序时返回错误，不再返回额外的魔法整数。

其他聚合行为：

- `GetMaxVersion`、`GetMinVersion` 返回输入中的原始字符串；等价版本保留最先出现者；
- `Sort` 稳定升序并原地修改输入；
- `GetClosestVersion` 使用主版本数字段的加权距离，目标位于范围外时取边界，等距时取较高版本；
- 空的最小值、最大值或最近值列表返回 `ErrEmptyVersionList`，空切片排序成功。

## 普通版本规则

普通规则用于现实漏洞数据中的宽松组件版本，不等同于严格 SemVer：

1. 去除首尾空白并转换为小写；`_` 转换为 `.`，`~` 和 `+` 转换为 `-`，兼容输入中的 `x86_64` 会被剔除。
2. 可从 `v1.2.3`、`OpenSSL_1_1_1d`、`@scope/package@2.0.0` 等标签中提取最可信的数字版本段。
3. 数字段以任意精度比较，忽略前导零；缺失的尾部零段等价，因此 `7.1 == 7.1.0 == 7.1.0.0`。
4. 数字主版本后的字母补丁号独立比较，因此 `1.1.1 < 1.1.1a < 1.1.1t`，无需 OpenSSL 专用分支。
5. 同一主版本下，正式版高于带后缀版本；后缀拆分为数字和文本 token 比较。
6. `MatchMain` 只忽略预发布与附加修订，仍会比较字母补丁号。
7. 纯字母数字标签和 40 字符 commit ID 属于不透明版本：精确范围可以判断相等或不相等；需要建立大小顺序的 `Compare`、排序和聚合操作会对不同标签返回 `ErrIncomparableVersion`。

根包和 `versionrange` 的 normal 均按通用业务规则处理 `+` 后缀，不保证严格 SemVer 的 build metadata 等价关系；这是 v1.1.1 对普通分类统一后的明确行为。

## 操作系统软件包匹配

导入独立子包：

```go
import "github.com/yanqin001/version-compare/osversion"
```

### 发行版算法族

`osversion` 根据 `Distribution` 分派算法。发行版名称会忽略大小写及常见分隔符，并识别常用别名。

| 算法族 | 发行版或选择器示例 | 版本语义 |
| --- | --- | --- |
| RPM | `rpm`、RHEL、CentOS、Fedora、Anolis、openEuler、SUSE、Rocky、AlmaLinux、Oracle Linux、Amazon Linux、Alibaba Cloud Linux | RPM Epoch-Version-Release / `rpmvercmp` |
| Debian | `debian`、`dpkg`、Ubuntu、Mint、Kali、Raspbian、Deepin、UOS | dpkg epoch、upstream version、debian revision |
| APK | Alpine、Wolfi、Chainguard、`apk` | apk-tools 版本、suffix 与 `-rN` revision |
| ALPM | Arch Linux、Manjaro、EndeavourOS、`alpm`、`pacman` | libalpm epoch、pkgver、pkgrel |
| Generic | 空值、`os`、`generic` 或未知名称 | 兼容旧 OS 数据的宽松主版本/后缀算法 |

公开的 `Distribution*` 常量覆盖表中列出的主流发行版；字符串形式的常见别名同样受支持。原生比较行为主要依据以下上游规范或参考实现独立实现：

- [RPM Spec：Version、Release 与 Epoch](https://rpm.org/docs/4.20.x/manual/spec.html#preamble-tags)
- [Debian Policy：Version 字段与比较规则](https://www.debian.org/doc/debian-policy/ch-controlfields.html#version)
- [apk-tools `version.c`](https://github.com/alpinelinux/apk-tools/blob/master/src/version.c)
- [ALPM `pkgver` 规范](https://alpm.archlinux.page/specifications/alpm-pkgver.7.html)

传入算法选择器（如 `rpm`）允许同算法族发行版；传入具体发行版（如 `anolis`）则要求候选发行版与其规范名称相符。候选 `Package.Distribution` 为空时视为未提供发行版信息，不额外拒绝。

未知发行版不会被猜测成 RPM 或 Debian，而是回退 generic。generic 会忽略 `/architecture`，按任意精度点分主版本和旧后缀规则比较；若已知真实发行版，请显式传入，避免得到与包管理器不同的顺序。

### `Package` 与 `Constraint` 联合匹配

```go
matcher, err := osversion.Compile(osversion.Constraint{
    Distribution:        osversion.DistributionAnolis,
    VersionExpression:   "(,255-10)",
    OSVersionExpression: "[23,24)",
    Architectures:       []string{"aarch64"},
})
if err != nil {
    return err
}

affected, err := matcher.Match(osversion.Package{
    Distribution: osversion.DistributionAnolis,
    Version:      "252.4-6.an23/aarch64",
})
```

以上 Anolis 示例返回 `true`。Matcher 会从旧格式中拆出：

- 软件包版本：`252.4-6.an23`；
- OS 版本：`an23`，按 Anolis 规则规范化为 `23`；
- 架构：`aarch64`。

显式设置 `Package.OSVersion` 或 `Package.Architecture` 时，显式值优先于从 `Version` 提取的值。联合匹配依次检查发行版、架构、OS 版本和软件包版本，任一条件不满足即返回 `false, nil`。`Constraint.VersionExpression` 必填；`OSVersionExpression` 和 `Architectures` 为空表示不限制对应维度。

低频调用也可使用无状态服务：

```go
svc := osversion.NewService()
result, err := svc.Compare(osversion.DistributionUbuntu, "1:2.0~rc1-1", "1:2.0-1")
```

OS `Compare` 同样只返回 `0/1/2`。

### 架构规则

架构约束在编译时规范化并复制：

- `nil`、空列表、单独的 `*` 或 `any` 表示不限制架构；
- 通配值不能与具体架构混用，空白约束项也是错误；
- 忽略大小写、首尾空白和前导 `/`；
- `x86_64 == amd64`、`aarch64 == arm64`、`ppc64el == ppc64le`；
- `noarch == all`，但只与彼此匹配，不是通配符；
- `src == source`；为迁移 Anolis 旧数据，空的候选架构也可匹配 `src`；
- 其他架构名称执行规范化后的精确匹配。

可单独编译并并发复用架构 Matcher：

```go
architectures, err := osversion.CompileArchitectures([]string{"amd64", "arm64"})
if err != nil {
    return err
}
fmt.Println(architectures.Match("/x86_64")) // true
```

## 错误与日志

根包公开以下可使用 `errors.Is` 判断的错误：

- `ErrEmptyVersion`
- `ErrInvalidVersion`
- `ErrInvalidExpression`
- `ErrUnsupportedVersionType`
- `ErrEmptyVersionList`
- `ErrIncomparableVersion`

`osversion` 提供对应的 `ErrEmptyVersion`、`ErrInvalidVersion`、`ErrInvalidExpression`，架构错误使用 `ErrInvalidArchitecture`。

`VersionSvc`、根包 `Compile`/`Matcher.Match`、`osversion.Service` 以及 OS 联合 `Compile`/`Matcher.Match` 在返回错误前会通过 Go 标准 `log` 包打印一次诊断信息。普通的“不匹配”不会打印日志。调用方仍需检查错误，不应通过日志文本判断错误类型。

低层的 `CompileArchitectures` 和包级 `MatchArchitecture` 只返回错误，不主动记录日志；若希望统一标准日志，请通过 `osversion.Service.MatchArchitecture` 调用。宿主程序可以使用标准库的 `log.SetOutput`、`log.SetFlags` 等配置日志目标与格式。

## 与旧 `Update` 字段的差异

新 API 不包含旧 CPE `Update` 字段，也不会通过子串猜测 `rc1`、`sp1`、`update19` 等更新标识。数据导入层应先把 `Version` 与 `Update` 合并成完整版本，再生成明确的精确值、列表或范围表达式；例如 `Version=6.7`、`Update=rc1` 应形成精确表达式 `6.7-rc1`。

- `Update="*"` 表示无需额外约束；
- `Update="-"` 等来源特有语义应在导入层转换成明确版本范围；
- 同一范围包含多个 Update 时，应展开成精确值列表或多个区间的并集。

该转换依赖漏洞数据源的字段定义，故不放入通用比较库。迁移时不要简单丢弃非 `*` 的 Update，否则会扩大漏洞影响范围。

## 性能与并发

- 版本与表达式使用单遍扫描解析，不依赖运行时正则或第三方库；
- Matcher 会预编译边界、分支修复版本、发行版原生版本、算法分派和架构集合；OS 候选在单次匹配中只解析一次；
- `VersionSvc` 与 `osversion.Service` 无状态，可并发使用；
- 根包 Matcher、OS Matcher 和 ArchitectureMatcher 编译后不可变，可跨 goroutine 共享；
- 排序、最小值、最大值和最近值操作会先解析每个输入一次，避免在 `O(n log n)` 比较中重复解析；
- 包不维护无界全局缓存；高频路径应由调用方持有并复用 Matcher。

提交性能相关改动时，请同时提供 benchmark，并使用 `go test -race ./...` 验证并发安全。

## 开源与贡献

项目使用 [MIT License](LICENSE)。贡献流程见 [CONTRIBUTING.md](CONTRIBUTING.md)，安全问题请按 [SECURITY.md](SECURITY.md) 私下报告。
