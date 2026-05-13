# excelexporter
适用于游戏项目的Excel配置表导出工具

## Excel导出
- 数据结构定义在proto文件中
- 解析proto文件,获取proto中的message的结构信息
- 解析Excel配置表,列名就是proto中定义的message的字段名
- 导出为proto对应的json或pb格式(protobuf序列化后的二进制数据,以便于更高效的加载)
- 支持批量导出,在一个excel里配置所有需要导出的配置表,可以批量导出并生成加载代码,把加载代码放到项目中,
可以一个接口就完成加载所有数据,并支持并发,热更新,增量加载
- 支持配置表关联检查
- 支持不同的配置表合并导出到同一个文件里
- 测试用例在tool/export_test.go

## 项目导入
- 加载导出的json或pb数据,直接反序列化成proto的message对象
- 测试用例在example/import_test.go

## 命令行
```shell
excelexporter -config=.\exporter.yaml
```
config: 配置文件
```yaml
#Excel导入目录(excel所在目录)
DataImportPath: "./data/excel"

#导出格式: json、pb
ExportFormats:
  - "json"
  - "pb"

#数据导出目录,和ExportFormats一一对应
DataExportPath:
  - "./data/json"
  - "./data/pb"

#可选项:导出md5文件完整路径,和ExportFormats一一对应
Md5ExportPath:
  - "./data/json/md5.json"
  - "./data/pb/md5.json"

#proto所在目录
ProtoPath: "./proto"

#需要解析的proto文件
ProtoFiles:
  - "export.proto"
  - "cfg.proto"

#代码模板目录
CodeTemplatePath: "./template/"

#代码模板
CodeTemplateFiles:
  - "data_mgr.go.template"

#代码导出目录 NOTE:和CodeTemplateFiles的数量要一致
CodeExportFiles:
  - "./cfg/data_mgr.go"

#导出分组标记 c s cs
ExportGroup: "s"

#默认的分组标记
DefaultGroup: "cs"

#导出总表的文件名
ExportAllExcelFile: "all.xlsx"

#导出总表的sheet名
ExportAllSheet: "ExportCfg"
```

## 导出总表(all.xlsx)
all.xlsx是一个导出总表(也叫索引表/配置注册表),它本身不包含具体的游戏配置数据,而是作为一个"元数据表",
记录了所有需要导出的Excel配置表的信息。程序通过解析这个总表,自动遍历并导出所有引用的配置Excel文件。

总表由配置文件中的`ExportAllExcelFile`和`ExportAllSheet`指定文件名和Sheet名,默认为`all.xlsx`中的`ExportCfg` Sheet。

### 总表的列定义
每一行代表一个配置表的导出定义,支持以下列:

| 列名 | 必填 | 说明 |
|------|------|------|
| Excel | 是 | 对应的Excel文件名(如`itemcfg.xlsx`) |
| Sheet | 是 | Excel中的Sheet名 |
| Message | 否 | 对应的protobuf Message名,不填则默认使用Sheet名 |
| Group | 否 | 分组标记(c/s/cs),用于按服务端/客户端筛选导出 |
| MgrType | 否 | 管理器类型: map(默认)/slice/object,详见下方说明 |
| MapKey | 否 | MgrType=map时的key字段名,不填则使用第一个非注释列 |
| CodeComment | 否 | 代码注释 |
| Merge | 否 | 合并名称,用于将多个Sheet的数据合并到同一个导出文件 |

### 总表Excel示例
```
---------------------------------------------------------------------------------------------
| Excel          | Sheet          | Message       | Group | MgrType | MapKey | CodeComment |
---------------------------------------------------------------------------------------------
| itemcfg.xlsx   | ItemCfg        | ItemCfg       | cs    | map     | CfgId  | 物品配置     |
---------------------------------------------------------------------------------------------
| levelcfg.xlsx  | LevelExp       | LevelExp      | cs    | slice   |        | 等级经验     |
---------------------------------------------------------------------------------------------
| global.xlsx    | GlobalCfg      | GlobalCfg     | cs    | object  |        | 全局配置     |
---------------------------------------------------------------------------------------------
| questcfg.xlsx  | QuestCfg       | QuestCfg      | cs    | map     | CfgId  | 任务配置     |
---------------------------------------------------------------------------------------------
```

### 工作流程
1. 程序读取all.xlsx总表,解析每行注册信息
2. 根据Group列和配置文件中的ExportGroup进行分组过滤
3. 打开每行Excel列指定的Excel文件,按Sheet列读取数据
4. 根据MgrType将数据转换为对应格式(map/slice/object)
5. 导出为JSON/PB格式文件
6. 根据代码模板生成数据管理器代码
7. 执行引用检查(Ref Check)

## 管理器类型(MgrType)
配置表支持3种管理器类型,通过总表的`MgrType`列指定:

### map(默认)
- 以某一列的值作为key,构建`map[key]RowData`的数据结构
- 适用于有唯一标识(如CfgId)的配置表,如物品配置、任务配置等
- 需要通过MapKey列指定key字段名(不填则默认使用第一个非注释列)
- 导出的JSON格式为`{"1": {...}, "2": {...}}`

### slice
- 所有行数据按顺序组成一个数组`[]RowData`
- 不需要key列,每一行直接作为数组的一个元素
- 适用于没有自然主键的列表配置,如等级经验表
- 导出的JSON格式为`[{...}, {...}]`

### object
- 采用Key-Value模式,Excel表中必须有`key`列和`value`列
- key列填写proto字段名,value列填写对应的值
- 最终导出为一个单独的JSON对象,而不是数组或map
- 适用于全局参数配置(如服务器参数、系统常量等),只有一"组"数据,用Key-Value方式编辑更直观

三种类型对比:
| 特性 | map | slice | object |
|------|-----|-------|--------|
| JSON输出 | `{"1": {...}}` | `[{...}]` | `{field: value}` |
| 需要key列 | 是 | 否 | 是(key=value模式) |
| 典型场景 | 有唯一ID的配置表 | 无自然主键的列表配置 | 全局参数配置 |

## 简单示例1:
- 由于proto文件中已经定义了数据结构,所以excel里只需要列名和proto定义的字段名一致,就可以知道字段的类型信息,
所以excel里不需要再指定字段类型,假设物品的proto定义如下
```protobuf3
message Item {
  int32 Id = 1;
  int32 Type = 2;
  string Name = 3;
}
```
那么物品配置表的excel格式就可以这样:
```
-------------------------
| Id     | Type  | Name |
-------------------------
| 1      | 1     | 物品1 |
-------------------------
| 2      | 1     | 物品2 |
-------------------------
| 3      | 2     | 物品3 |
-------------------------
```

## 示例2:字段也是message结构
```protobuf3
message Test {
  int32 Id = 1;
  Item Item1 = 2;
  Item Item2 = 3;
  repeated Item Items = 4;
}
```
配置表的excel格式就可以这样:
```
-------------------------------------------------------------
| Id  | Item1      | Item2                 | Items          |
|     | #Field=no  | #Field=full           | #Field=Id_Type |
-------------------------------------------------------------
| 1   | 1_1_物品1   | Id_1#Type_1#Name_物品1 | 1_1;2_1        |
-------------------------------------------------------------
| 2   | 2_1_物品2   | Id_2#Type_1#Name_物品2 | 2_1;3_2        |
-------------------------------------------------------------
| 3   | 3_2_物品3   | Id_3#Type_2#Name_物品3 | 3_2;1_1        |
-------------------------------------------------------------
```
一个单元格要配置一个message的数据,就需要在一个单元格里填写多个字段的数据
- #Field=no   表示Item1不需要填写字段名,以_作为分隔符,按照字段顺序进行赋值,适用于字段少的结构简单的message,
缺点是兼容性差,当message的字段做了更新,可能导致解析异常
- #Field=full 表示Item2需要填写字段名,以Field1_v1#Field2_v2的格式,按照字段名进行赋值,填写麻烦但是兼容性强,
当message增删了字段或者调整了字段的顺序,也不影响字段的解析
- #Field=Field1_Field2_FieldN 是前2种格式的结合,既简洁又保留了兼容性,解析时会按照表头指定的字段名进行赋值,
且单元格不需要再每行填写字段名

## 示例3: 单元格使用json格式
```
-----------------------------------------------------------------------
| Id  | Item1      | Item2                           | Items          |
|     | #Field=no  | #Format=json                    | #Field=Id_Type |
-----------------------------------------------------------------------
| 1   | 1_1_物品1   | {"Id":1,"Type":1,"Name":"物品1"} | 1_1;2_1        |
-----------------------------------------------------------------------
| 2   | 2_1_物品2   | {"Id":2,"Type":1,"Name":"物品2"} | 2_1;3_2        |
-----------------------------------------------------------------------
| 3   | 3_2_物品3   | {"Id":3,"Type":2,"Name":"物品3"} | 3_2;1_1        |
-----------------------------------------------------------------------
```

## 示例4: 单元格关联检查
格式: #Ref=要关联检查的配置表名,假如有一个配置表Item,下面的Item2配置了#Ref=Item,在导表时,将会自动关联检查Item2中配置的Id是否在Item中存在,如果不存在,将会输出错误信息
```
-------------------------------------------------------------
| Id  | Item1      | Item2                 | Items          |
|     | #Field=no  | #Field=full#Ref=Item  | #Field=Id_Type |
-------------------------------------------------------------
| 1   | 1_1_物品1   | Id_1#Type_1#Name_物品1 | 1_1;2_1        |
-------------------------------------------------------------
| 2   | 2_1_物品2   | Id_2#Type_1#Name_物品2 | 2_1;3_2        |
-------------------------------------------------------------
| 3   | 3_2_物品3   | Id_3#Type_2#Name_物品3 | 3_2;1_1        |
-------------------------------------------------------------
```

## 示例5: 多列合并为数组 (#Merge)
对于proto中定义的repeated字段,支持在Excel中使用多列来配置,每列只编辑单个元素,导表时自动合并为数组。
格式: 列名#Merge,多列使用相同的列名即可自动合并。

例如proto定义如下:
```protobuf3
message AddElemArg {
  int32 CfgId = 1;  // 配置id
  int32 Num = 2;    // 数量
}

message QuestCfg {
  int32 CfgId = 1;
  repeated AddElemArg Rewards = 2; // 任务奖励
}
```

Excel配置格式:
```
---------------------------------------------------------------
| CfgId | Rewards         | Rewards         | Rewards         |
|       | #Merge#Field=no | #Merge#Field=no | #Merge#Field=no |
---------------------------------------------------------------
| 1     | 1_100           | 2_50              | 3_200         |
---------------------------------------------------------------
| 2     | 10_500          |                   | 20_100        |
---------------------------------------------------------------
```

导出后的JSON:
```json
{
  "CfgId": 1,
  "Rewards": [
    {"CfgId": 1, "Num": 100},
    {"CfgId": 2, "Num": 50},
    {"CfgId": 3, "Num": 200}
  ]
}
```

特点:
- 每个Rewards#Merge列只编辑单个AddElemArg元素
- 空单元格会被自动忽略,不参与合并
- 支持与#Field、#Format等其他标记组合使用
- 适用于repeated字段元素较多、结构较复杂的场景

## 示例6: 展开字段(点号语法)
当proto字段本身是message时,支持将子字段拆分为多列配置,列名格式为`父字段.子字段`,导出时自动合并为父message。

例如proto定义如下:
```protobuf3
message BaseInfo {
  int32 CfgId = 1;
  int32 Type = 2;
}

message ComboClass {
  int32 Id = 1;
  string Name = 2;
  BaseInfo Base = 3;
}
```

Excel配置格式:
```
-------------------------------------------
| Id | Name    | Base.CfgId | Base.Type   |
-------------------------------------------
| 1  | ComboA  | 1001       | 1           |
-------------------------------------------
| 2  | ComboB  | 1002       | 2           |
-------------------------------------------
```

## 示例7: 分组导出(##group)
可在表中增加`##group`行控制列导出分组,并结合配置文件中的`ExportGroup`和`DefaultGroup`生效。

Excel配置格式:
```
------------------------------------------------
| ##var   | Id  | Name | ServerOnly | ClientOnly |
------------------------------------------------
| ##group | cs  | cs   | s          | c          |
------------------------------------------------
|         | 1   | A    | 100        | icon_a     |
------------------------------------------------
```

说明:
- 当`ExportGroup=s`时,会导出`Id/Name/ServerOnly`
- 当`ExportGroup=c`时,会导出`Id/Name/ClientOnly`
- 当列未显式配置分组时,使用`DefaultGroup`

## 示例8: 显式列定义行(##var)
当首行不是字段名(例如有说明或注释)时,可以用`##var`明确指定“这一行是列定义行”。

Excel配置格式:
```
-----------------------------------------
| # 这是说明行,不会参与数据解析            |
-----------------------------------------
| ##var   | Id | Type | Name            |
-----------------------------------------
|         | 1  | 1    | 物品1            |
-----------------------------------------
```

## 示例9: 分隔符规则与json便捷写法
常见分隔规则:
- repeated/map第一层元素: 使用`;`或换行分隔
- 嵌套repeated/map元素: 使用`,`分隔(避免和外层冲突)

示例:
```
Elems: 1_10;2_20;3_30
Nested: 1,2,3
```

`#Format=json`示例:
```
---------------------------------------
| Item                | Item           |
| #Format=json        | #Format=json   |
---------------------------------------
| {"Id":1,"Type":2}   | "Id":2,"Type":3 |
---------------------------------------
```

说明:
- 建议优先使用标准JSON(带`{}`),可读性更好
- 在支持的场景下可省略最外层`{}`

## 示例10: 自定义分隔符(#Sep)
默认情况下,message字段值之间使用`_`作为分隔符(如`1_100`),但当字段值本身包含`_`时(如物品名称),会导致解析错误。可以通过`#Sep`指定自定义分隔符来解决。

格式: `#Sep=分隔符`,例如`#Sep=|`表示使用`|`作为分隔符。

proto定义:
```protobuf3
message ItemNum {
  int32 CfgId = 1;
  int32 Num = 2;
}

message QuestCfg {
  int32 CfgId = 1;
  repeated ItemNum Rewards = 2;
}
```

### #Merge + 自定义分隔符
```
---------------------------------------------------------
| CfgId | Rewards             | Rewards             |
|       | #Merge#Sep=|        | #Merge#Sep=|        |
---------------------------------------------------------
| 1     | 1|100               | 2|200               |
---------------------------------------------------------
| 2     | 3|50                | 4|300               |
---------------------------------------------------------
```

导出后的JSON:
```json
[
  {"CfgId": 1, "Rewards": [{"CfgId": 1, "Num": 100}, {"CfgId": 2, "Num": 200}]},
  {"CfgId": 2, "Rewards": [{"CfgId": 3, "Num": 50}, {"CfgId": 4, "Num": 300}]}
]
```

说明:
- `#Sep`仅影响当前列的第一层字段分隔符,嵌套层级仍使用默认的`_`分隔符
- 适用于字段值中包含`_`的场景,避免解析歧义
- 可与`#Field`、`#Merge`等标记自由组合使用

## 示例11: slice格式配置表
当配置表没有自然的唯一主键时,可以使用slice格式,所有行数据按顺序组成一个数组。

proto定义:
```protobuf3
message LevelExp {
  int32 Level = 1;   // 等级
  int32 NeedExp = 2; // 升到该等级需要的经验值
}
```

在all.xlsx总表中注册(MgrType填写slice):
```
---------------------------------------------------------
| Excel         | Sheet    | Message | MgrType | MapKey |
---------------------------------------------------------
| levelcfg.xlsx | LevelExp | LevelExp| slice   |        |
---------------------------------------------------------
```

levelcfg.xlsx中的LevelExp Sheet配置:
```
----------------------------
| Level | NeedExp          |
----------------------------
| 1     | 0                |
----------------------------
| 2     | 100              |
----------------------------
| 3     | 300              |
----------------------------
| 4     | 600              |
----------------------------
| 5     | 1000             |
----------------------------
```

导出后的JSON(LevelExp.json):
```json
[
  {"Level": 1, "NeedExp": 0},
  {"Level": 2, "NeedExp": 100},
  {"Level": 3, "NeedExp": 300},
  {"Level": 4, "NeedExp": 600},
  {"Level": 5, "NeedExp": 1000}
]
```

说明:
- slice格式不需要指定MapKey,每行直接作为数组的一个元素
- 所有特性(#Field、#Format、#Merge、#Sep、展开字段等)均适用于slice格式
- slice格式同样支持分组导出(##group)

## 示例12: object格式配置表(Key-Value模式)
object格式采用Key-Value模式,适用于全局配置(如服务器参数、系统常量等)。
Excel表中必须有`key`列和`value`列,key列填写proto字段名,value列填写对应的值。

proto定义:
```protobuf3
message GlobalCfg {
  string ServerName = 1;   // 服务器名称
  int32 MaxLevel = 2;      // 最大等级
  int32 MaxPlayerNum = 3;  // 最大玩家数
  float DropRate = 4;      // 掉落倍率
  bool EnablePvP = 5;      // 是否开启PvP
}
```

在all.xlsx总表中注册(MgrType填写object):
```
---------------------------------------------------------
| Excel        | Sheet     | Message  | MgrType | MapKey |
---------------------------------------------------------
| global.xlsx  | GlobalCfg | GlobalCfg| object  |        |
---------------------------------------------------------
```

global.xlsx中的GlobalCfg Sheet配置:
```
-------------------------------
| key          | value        |
-------------------------------
| ServerName   | 测试服务器    |
-------------------------------
| MaxLevel     | 100          |
-------------------------------
| MaxPlayerNum | 5000         |
-------------------------------
| DropRate     | 1.5          |
-------------------------------
| EnablePvP    | true         |
-------------------------------
```

导出后的JSON(GlobalCfg.json):
```json
{
  "ServerName": "测试服务器",
  "MaxLevel": 100,
  "MaxPlayerNum": 5000,
  "DropRate": 1.5,
  "EnablePvP": true
}
```

说明:
- key列填写proto中定义的字段名,value列填写该字段的值
- 程序会根据proto定义自动识别value列的数据类型(int/string/float/bool等)
- value列支持复杂类型(message、repeated、map等),可配合#Field、#Format等标记使用
- object格式也支持展开字段(点号语法),如key列填写`Base.CfgId`可将子字段合并到父message中
- 导出的JSON是一个单独的对象(不是数组或map)
