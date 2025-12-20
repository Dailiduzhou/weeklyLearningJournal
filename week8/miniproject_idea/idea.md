# `idea`
## 起因
木犀后端组实习期间，有个项目叫做**木犀骇客**。
本意是想让我们熟悉用`Go`语言写http请求。
但我有一个奇特的想法，
既然已经有现成的网站源码了，
我们能不能基于此，
写一个用**图形化界面**模拟黑客入侵银行的**网页应用**。
## 构想
木犀骇客每步的输出（示例）
总体需要调用的功能：
1. 发送GET、POST的http请求。
2. 对http请求的header、payload内容进行编辑。
3. 获取`base64`编码的虹膜图片并解码，存储在用户本地，并上传图片（按路径或默认）。
4. `base64`解码,`AES`编码。

## 游戏流程
![pic1](../images/pic1.png)
![pic2](../images/pic2.png)
![pic3](../images/pic3.png)
---
`Step1`

```
Send request successfully! Please check your response body.
response header:
X-Frame-Options : DENY
X-Request-Id : bc57dbdd-bdaf-4229-b0d4-b93790ac7c04
X-Xss-Protection : 1; mode=block
Access-Control-Allow-Origin : *
Expires : Thu, 01 Jan 1970 00:00:00 GMT
Last-Modified : Wed, 03 Dec 2025 08:59:49 GMT
Map-Fragments : muxi
X-Content-Type-Options : nosniff
Date : Wed, 03 Dec 2025 08:59:49 GMT
Content-Length : 356
Cache-Control : no-cache, no-store, max-age=0, must-revalidate, value
Content-Type : application/json; charset=utf-8
Passport : eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjb2RlIjoiT01FTiIsImlhdCI6MTc2NDc1MjM4OSwibmJmIjoxNzY0NzUyMzg5fQ.GrpXNuRZ94ZjE9ip6jIegaPBAd-x6tvIQ2ALi35wKhg
Message:
response body:
1.Message:
1.Message:
OK
1.Message:
OK
2.Text:
访问成功后，网站会给你返回信息，在header中找到你的passport。
将passport加入到你以后的每次请求头中。
完成上述步骤后，用代码访问 http://http-theft-bank.gtainccnu.muxixyz.com/api/v1/organization/secret_key，注意查收其中response的信息。
3.ExtraInfo:
```

---

`step2`

```
1.Message:
OK
2.Text:
恭喜你拿到了 passport，现在你可以着手准备骇入银行。
银行的第一道门是代码安全门，我们计划将错误代码写入此门来破解它。
但是这个门具有识别明文代码的功能，所以我们还需要一个密钥加密我们的错误代码，再写入至此门。
不需要担心，两者我们都为你提供了，你只需要解析 response 中的密文（在 ExtraInfo 中）来得到它们。
你现在的任务：
解析密文，获取 error_code 和 secret_key
使用 secret_key 加密 error_code
然后将加密过的 error_code 放入 请求body 并以 "正确的请求方法" 发送至 http://http-theft-bank.gtainccnu.muxixyz.com/api/v1/bank/gate , 同时注意response的信息。
3.ExtraInfo:
c2VjcmV0X2tleTpNdXhpU3R1ZGlvMjAzMzA0LCBlcnJvcl9jb2RlOmZvciB7Z28gZnVuYygpe3RpbWUuU2xlZXAoMSp0aW1lLkhvdXIpfSgpfQ==

secret_key:MuxiStudio203304, error_code:for {go func(){time.Sleep(1*time.Hour)}()}


encryption : jwAaUqIJzyrWc3eAol4U8hLC4gql7FfdBNDxn6nBvOLoyxTtlTJFs9Izv8iIbJWt
```
---
`step3`
```
1.Message:
OK
2.Text:
干的漂亮！你已经突破了第一扇门，请继续访问 http://http-theft-bank.gtainccnu.muxixyz.com/api/v1/bank/iris_recognition_gate 。
3.ExtraInfo:
```
---
`step4`
```
Message:
response body:
1.Message:
OK
2.Text:
你现在已经到第二扇门了，是虹膜识别安全门。
你需要向组织请求已准备好的虹膜样本，访问 http://http-theft-bank.gtainccnu.muxixyz.com/api/v1/organization/iris_sample 下载图片。
再将此图片上传至 http://http-theft-bank.gtainccnu.muxixyz.com/api/v1/bank/iris_recognition_gate 以破解此门，加油！
3.ExtraInfo:
```
---
`step5`
```
Message:
response body:
1.Message:
OK
2.Text:
还剩最后一道门了！
我们需要银行结构图碎片，这些碎片就隐藏在前面某四个路由的响应头中，位于 map-fragments 字段。
将它们用"/"拼起来就是最后一道门的所在位置！注意response的信息。
3.ExtraInfo:
```
---
`step6`
```
response body:
1.Message:
OK
2.Text:
OMEN，真亏你能来到这里！在你眼前的就是最后的密码门了。
但是密码位数未知，看来只能通过全排列程序去暴力破解。

>示例如下：
============================================
输入：
3
1 2 3
输出：
[[1 2 3][1 3 2][2 1 3][2 3 1][3 1 2][3 2 1]]
============================================

>代码模板:

func permute(nums []int) [][]int {
    // insert your code

}

func main() {
    var n int
        fmt.Scanf("%d", &n)

        testSlice := make([]int, n)
    // 标准输入n个不重复的数字

    res := permute(testSlice)
    fmt.Println(res)
}

请在完成此程序后上传至 http://http-theft-bank.gtainccnu.muxixyz.com/api/v1/muxi/backend/computer/examination 来破解最后的密码门
3.ExtraInfo:
```

这里的判断代码是否能实现全排列的功能有些复杂，考虑改掉。
---
`step7`
```
Message:
response body:
1.Message:
OK
2.Text:
END
我就知道你能成功！Backend组织欢迎你！
3.ExtraInfo:
```
