<div align="center">

<img src="docs/assets/vpnban-banner.svg" alt="vpnban — نگهبان بررسی سرورهای VPN" width="100%">

# vpnban · بان

**نگهبان بررسی‌های VPN. پروتکل را انتخاب کنید و نتیجه را به خاطر بسپارید.**

[English](README.md) · [فارسی](README.fa.md)

[![CI](https://github.com/WhoisGray/vpnban/actions/workflows/ci.yml/badge.svg)](https://github.com/WhoisGray/vpnban/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/WhoisGray/vpnban?color=38bdf8)](https://github.com/WhoisGray/vpnban/releases/latest)
[![Contributions welcome](https://img.shields.io/badge/contributions-welcome-34d399)](#مشارکت)

</div>

---

**vpnban** (بان، «نگهبان») ابزار خط فرمان Go است که سرورها و نتیجه‌های بررسی را به خاطر می‌سپارد. فهرست سرورها را از API رسمی NordVPN می‌گیرد یا فهرست دامنه و IPv4 شما را می‌خواند و نتیجه را در JSON، CSV و Excel نگه می‌دارد.

> نتیجهٔ بررسی، وضعیت پاسخ‌گویی از شبکهٔ فعلی شماست و ورود موفق، ساخت تونل یا سرعت اینترنت را تضمین نمی‌کند. فقط سرورهایی را بررسی کنید که اجازهٔ آزمایششان را دارید.

## چرا بان؟

- انتخاب IKEv2، WireGuard، OpenVPN یا ترکیبی از آن‌ها در حالت NordVPN.
- فیلتر کشور و بار سرور و محدود کردن تعداد بررسی‌ها.
- پشتیبانی از فهرست شخصی و نتیجه‌های ذخیره‌شده.
- خروجی قابل استفاده برای تحلیل و اشتراک‌گذاری.
- بسته‌های macOS، Linux و Windows برای amd64 و arm64.

## شروع سریع

پس از نصب، یک بررسی محدود انجام دهید:

```bash
vpnban -version
vpnban -nord -country se -limit 10
```

این دستور IKEv2 را روی حداکثر ده سرور سوئد بررسی می‌کند. در PowerShell، اگر فایل در پوشهٔ فعلی است، از `.\vpnban.exe` استفاده کنید.

## دانلود

آخرین نسخه را با یک دستور نصب کنید. دستور خودش سیستم و معماری را می‌فهمد و همان بسته را با چک‌سام SHA256 می‌گذارد روی PATH.

macOS و Linux. همین یک خط نصب می‌کند، و با `-update` همان خط را برای آخرین نسخه دوباره می‌زند:

```bash
curl -fsSL https://raw.githubusercontent.com/WhoisGray/vpnban/refs/heads/main/install.sh | sh -s -- -update
```

Windows در PowerShell. دوباره اجرا کردنش یعنی به‌روزرسانی:

```powershell
irm https://raw.githubusercontent.com/WhoisGray/vpnban/refs/heads/main/install.ps1 | iex
```

برای به‌روزرسانی نسخه‌های دارای پرچم `-update`، روی هر سه سیستم:

```bash
vpnban -update
```

نسخهٔ قدیمی `v1.0.1` پرچم `-update` را ندارد. برای این نسخه، همان دستور نصب را دوباره اجرا کنید.

فایل هر سیستم از [Releases](https://github.com/whoisGray/vpnban/releases) هم جداگانه هست. با پوش کردن یک تگ مثل `v1.1.0` ساخته می‌شود. کنار فایل‌ها `checksums.txt` هست.

| سیستم | معماری | فایل |
| --- | --- | --- |
| macOS اپل سیلیکون | arm64 | `vpnban_1.1.0_darwin_arm64.tar.gz` |
| macOS اینتل | amd64 | `vpnban_1.1.0_darwin_amd64.tar.gz` |
| Linux | amd64 | `vpnban_1.1.0_linux_amd64.tar.gz` |
| Linux | arm64 | `vpnban_1.1.0_linux_arm64.tar.gz` |
| Windows | amd64 | `vpnban_1.1.0_windows_amd64.zip` |
| Windows | arm64 | `vpnban_1.1.0_windows_arm64.zip` |

`1.1.0` همان شمارهٔ تگ است، بدون `v`.

### macOS و Linux

```bash
tar -xzf vpnban_1.1.0_darwin_arm64.tar.gz
sudo mv vpnban /usr/local/bin/
vpnban -version
```

بایینری امضا نشده است. اگر Gatekeeper روی مک جلوی اجرا را گرفت:

```bash
xattr -d com.apple.quarantine /usr/local/bin/vpnban
```

جمع چک:

```bash
shasum -a 256 -c checksums.txt
```

روی Linux به جای `shasum` از `sha256sum -c checksums.txt` استفاده کنید.

### Windows

zip را باز کنید و در PowerShell:

```powershell
.\vpnban.exe -version
Get-FileHash .\vpnban_1.1.0_windows_amd64.zip -Algorithm SHA256
```

هش را با خط همان فایل در `checksums.txt` مقایسه کنید.

### نصب با Go

Go 1.22 یا جدیدتر:

```bash
go install github.com/whoisGray/vpnban@latest
```

## استفاده

```bash
vpnban -nord                         # فقط IKEv2
vpnban -nord -proto wire             # فقط WireGuard
vpnban -nord -proto ikev2,wire
vpnban -nord -proto full             # همه‌ی پروتکل‌ها
vpnban -nord -country se -limit 40 -c 64
vpnban -all-providers -country se -limit 10 -c 32
vpnban -file servers.json
vpnban -file server.txt              # هر خط یک دامنه یا IPv4
```

`-proto` این‌ها را می‌پذیرد: `ikev2`، `wire` (همان `wg` و `wireguard`)، `openvpn`، فهرست با ویرگول، یا `full`.

`full` علاوه بر WireGuard و OpenVPN و IKEv2، پورت‌های udp/53 و tcp/80 را هم چک می‌کند تا پاسخ عجیب فایروال در جدول بیاید. اسکن تک‌پروتکل این دو پورت اضافه را نمی‌زند.

سروری که پروتکل انتخاب‌شده را ندارد از فهرست حذف می‌شود.

خروجی Nord این سه فایل است:

- `nord-results.json`
- `nord-results.csv`
- `nord-results.xlsx`

پرچم `-all-providers`، NordVPN و همهٔ ارائه‌دهندگان پشتیبانی‌شدهٔ gluetun را در یک اجرا بررسی می‌کند. پیش‌فرض این حالت IKEv2، WireGuard و OpenVPN است. سقف `-limit` برای هر ارائه‌دهنده جداگانه اعمال می‌شود و ظرفیت `-c` بین آن‌ها مشترک است. خروجی‌های ترکیبی `all-providers-results.json`، `.csv` و `.xlsx` ستون `provider` دارند. اگر بررسی یک ارائه‌دهنده خطا بدهد، نتیجه‌های موجود ذخیره می‌شوند و برنامه با وضعیت خطا خارج می‌شود.

حالت فایل، خروجی را در `vpnban-results.json` و `vpnban-results.csv` و `vpnban-results.xlsx` می‌نویسد.

کش فهرست Nord در `~/.cache/nordike` است و ۶ ساعت می‌ماند. `-refresh` کش را دور می‌زند.

اگر VPN روشن باشد و DNS فقط آدرس خصوصی (مثل `10.10`) برگرداند، فاز کشف متوقف می‌شود. VPN را خاموش کنید، یا با `-allow-poisoned` ادامه دهید.

### منوی تعاملی ترمینال

```bash
vpnban interactive
```

با کلیدهای جهت و Enter می‌توانید بررسی ارائه‌دهندگان یا فهرست شخصی، ساخت پروفایل اپل، مدیریت حساب‌ها و تنظیمات را انتخاب کنید. وقتی ورودی یا خروجی به ترمینال وصل نباشد، منو با شماره کار می‌کند. در منوی بررسی، سقف اولیه برای هر ارائه‌دهنده ۱۰ سرور است.

### ساخت پروفایل اپل

۱. برای هر حساب یک نام مستعار جدا بگذارید. چند حساب از یک ارائه‌دهنده هم پشتیبانی می‌شود. هنگام وارد کردن رمز در ترمینال، رمز نمایش داده نمی‌شود.

```bash
vpnban accounts add -name nord-home -provider nordvpn
vpnban accounts add -name nord-work -provider nordvpn
vpnban accounts list
```

۲. سرورها را با IKEv2 بررسی کنید و از گزارش JSON پروفایل بسازید.

```bash
vpnban -nord -proto ikev2 -limit 30
vpnban profile -account nord-home -input nord-results.json -output nord-home.mobileconfig
```

برای ساخت گزارش ترکیبی، `vpnban -all-providers -proto ikev2 -limit 30` را اجرا کنید و `all-providers-results.json` را به `-input` بدهید. هر پروفایل با حساب انتخابی ساخته می‌شود و تا هشت سرور از همان ارائه‌دهنده را که هر دو فاز IKEv2 را پاس کرده‌اند، به ترتیب کمترین تأخیر وارد می‌کند. سقف را با `-limit` عوض کنید.

۳. در مک می‌توانید ساختار فایل را با `plutil -lint nord-home.mobileconfig` بررسی کنید. برای نصب، فایل را باز کنید و در آیفون یا آیپد به **Settings → General → VPN & Device Management** و در مک به **System Settings → General → Device Management** بروید. [راهنمای نصب در مک](https://support.apple.com/guide/mac-help/configuration-profiles-standardize-settings-mh35561/mac)

ساخت پروفایل برای ارائه‌دهندگانی کار می‌کند که اتصال دستی IKEv2 با نام کاربری و رمز دارند و دست‌کم یک سرورشان هر دو فاز را پاس کرده است. اطلاعات ورود سرویس را از ارائه‌دهنده بگیرید. اگر گواهی ریشه لازم است، فایل رسمی آن را با `-ca` بدهید یا دستورالعمل اعتماد به گواهی را روی دستگاه دنبال کنید. اگر ارائه‌دهنده Remote ID دیگری تعیین کرده، آن را با `-remote-id` وارد کنید. [راهنمای IKEv2 نورد](https://support.nordvpn.com/hc/en-us/articles/20371657885329-How-to-connect-to-NordVPN-with-a-manual-IKEv2-IPSec-connection-on-an-iOS-device)

فایل حساب‌ها و پروفایل‌های خروجی حاوی اطلاعات ورود سرویس‌اند. آن‌ها را خصوصی نگه دارید. فهرست حساب‌ها فقط نام مستعار و ارائه‌دهنده را نشان می‌دهد.

### محل حساب‌ها و راهنمای فرمان‌ها

```bash
vpnban config path
vpnban config set-dir "$HOME/vpnban-accounts"
vpnban help profile
vpnban help accounts
vpnban help config
```

محل پیش‌فرض زیر پوشهٔ تنظیمات کاربر در سیستم‌عامل است. از **Settings** در منوی تعاملی یا `config set-dir` می‌توانید مسیر را عوض کنید. هنگام انتخاب پوشهٔ جدید، حساب‌ها کپی می‌شوند و فایل‌های قبلی باقی می‌مانند. با انتخاب پوشهٔ موجود، حساب‌های همان پوشه فعال می‌شوند. متغیر `VPNBAN_CONFIG_DIR` مسیر را برای یک اجرا تعیین می‌کند.

### معنی چند ستون

سرورهایی که هر دو فاز IKEv2 را پاس کرده‌اند، به ترتیب کمترین تأخیر در بالای گزارش می‌آیند. گزارش ارائه‌دهندگان از تأخیر IKEv2 استفاده می‌کند. در حالت فایل، اولویت با میانگین تأخیر TCP است و در نبود آن از تأخیر IKEv2 استفاده می‌شود.

| مقدار | معنی |
| --- | --- |
| `real` | پاسخ شبیه سرور واقعی همان پروتکل بود |
| `p1` | فقط فاز اول IKEv2 جواب داد |
| `silent` | پاسخی نیامد |
| `fresh=yes` | شواهد پروتکل پذیرفته شد؛ پاسخ فقط فاز اول IKEv2 هم ممکن است کافی باشد |
| `unproven` | جواب قابل اتکا نبود |

## پرچم‌ها

| پرچم | پیش‌فرض | کار |
| --- | --- | --- |
| `-nord` | خاموش | گرفتن فهرست NordVPN و آزمایش |
| `-provider` | | آزمایش یک ارائه‌دهنده از فهرست gluetun |
| `-all-providers` | خاموش | بررسی همهٔ ارائه‌دهندگان در یک گزارش |
| `-proto` | `ikev2`؛ برای همه `ikev2,wire,openvpn` | پروتکل آزمایش |
| `-country` | همه | کد یا نام کشور، با ویرگول |
| `-limit` | `0` | سقف تعداد برای هر ارائه‌دهنده؛ در NordVPN کم‌بارترین‌ها انتخاب می‌شوند |
| `-load` | `100` | رد کردن سرور شلوغ‌تر از این |
| `-offline` | خاموش | سرورهای غیرآنلاین هم بیایند |
| `-refresh` | خاموش | دانلود دوبارهٔ فهرست |
| `-ttl` | `6h` | عمر کش |
| `-c` | `80`؛ برای ارائه‌دهندگان `32` | تعداد کارهای هم‌زمان |
| `-ike-timeout` | `1s` | مهلت IKEv2 |
| `-tcp-timeout` | `1200ms` | مهلت TCP. در `-nord` اگر ندهید `1s` است |
| `-ike-user` | بسته به حالت | هویت کاربر در IKE_AUTH |
| `-ike-id` | بسته به حالت | شناسهٔ سمت سرور در IKE_AUTH |
| `-file` | | JSON کاتالوگ، فایل متنی، یا نتیجهٔ قبلی |
| `-phase` | | `1` کشف DNS، `2` آزمایش، `all` هر دو |
| `-dns` | `1.1.1.1:53` | سرور DNS فاز کشف |
| `-allow-poisoned` | خاموش | ادامه با وجود IP خصوصی |
| `-json` `-csv` `-xlsx` | بسته به حالت | مسیر خروجی |
| `-version` | | چاپ نسخه و خروج |
| `-update` | | جایگزینی همین برنامه با آخرین Release گیت‌هاب |

در `-nord` اگر `-ike-user` و `-ike-id` را ندهید، مقدارشان `health@nordvpn.com` و `nordvpn.com` است. در حالت فایل، `this is health check` و `ironnodes.com`.

## انتشار

```bash
git tag v1.0.0
git push origin v1.0.0
```

workflow به نام Release اول تست را اجرا می‌کند، بعد شش بسته را می‌سازد و روی صفحهٔ Releases می‌گذارد. `vpnban -version` همان تگ را چاپ می‌کند.

هر push و هر pull request هم با workflow به نام CI تست و کامپایل هر شش سیستم را می‌گیرد.

## ساخت محلی

```bash
go test ./...
go build -o vpnban .
./vpnban -version
```


## نکته‌های مهم

پرچم `-proto` پروتکل را در حالت‌های ارائه‌دهنده تعیین می‌کند. حالت فایل شخصی روند بررسی خودش را دارد. نتیجهٔ `silent` می‌تواند ناشی از فیلتر، قطعی مسیر یا کندی باشد.

نصاب‌ها معماری و سیستم را تشخیص می‌دهند و چک‌سام آرشیو را بررسی می‌کنند. می‌توانید پیش از اجرا، [نصاب Unix](install.sh) و [نصاب Windows](install.ps1) را بخوانید. اگر مسیر نصب روی `PATH` نبود، دستور چاپ‌شده توسط نصاب را اجرا کنید.

## نقشهٔ راه ارائه‌دهندگان

- [x] NordVPN: دریافت از API رسمی، فیلتر، کش و بررسی پروتکل‌ها.
- [x] فهرست دامنه و IPv4 شخصی و ورودی JSON پشتیبانی‌شده.
- [x] ۲۲ ارائه‌دهندهٔ دیگر: کشف از کاتالوگ gluetun و بررسی پروتکل‌ها با `-provider` یا `-all-providers`.
- [ ] رابط مشترک ارائه‌دهندگان برای جداسازی کشف، اطلاعات، کش و بررسی.
- [ ] تست با دادهٔ نمونه برای تجزیهٔ پاسخ، خطا و پروتکل پشتیبانی‌نشده.
- [ ] مستندات راه‌اندازی، محدودیت‌ها و نمونه‌ها به انگلیسی و فارسی.

ارائه‌دهندهٔ دیگری مدنظر دارید؟ [درخواست اضافه‌کردن ارائه‌دهنده](https://github.com/WhoisGray/vpnban/issues/new?title=Provider%20request%3A%20) را با نام سرویس، منبع مستند سرورها، پروتکل‌ها و نیاز به حساب ثبت کنید.

## مشارکت

برای اضافه‌کردن ارائه‌دهندگان، کدنویسی Go، تست، ترجمه و گزارش خطای قابل بازتولید کمک کنید.

1. یک Issue باز کنید یا کار موجودی را انتخاب کنید و محدودهٔ کارتان را توضیح دهید.
2. مخزن را Fork کنید و یک شاخهٔ مشخص بسازید.
3. برای ارائه‌دهندهٔ جدید، منبع سرورها، احراز هویت، محدودیت درخواست و پروتکل‌ها را مستند کنید. رمز، توکن و کلید خصوصی را در مخزن قرار ندهید.
4. تست تجزیهٔ پاسخ با دادهٔ نمونه و حالت‌های خطا اضافه کنید. از کد بررسی فعلی استفاده کنید و پروتکل‌های پشتیبانی‌نشده را واضح مشخص کنید.
5. هر دو README را به‌روز کنید، `go test ./...` و `go build -o vpnban .` را اجرا کنید و Pull Request را با نمونهٔ خروجی و نتیجهٔ بررسی‌ها بفرستید.

نقطه‌های شروع: [nord.go](nord.go) برای کشف و فیلتر، [main.go](main.go) برای خط فرمان و ورودی، و [ike.go](ike.go)، [wg.go](wg.go)، [ovpn.go](ovpn.go) برای بررسی پروتکل‌ها. رابط مشترک ارائه‌دهندگان هنوز جزو نقشهٔ راه است.

---

اگر بان برایتان مفید بود، با یک **ستاره ⭐** و اشتراک تجربه‌تان از پروژه حمایت کنید.
