# excelexporter
Excel配置表导出工具,仅仅是一个测试项目

Excel导出
- 数据结构定义在proto文件中
- 解析proto文件,获取proto中的message的结构信息
- 解析Excel配置表,列名就是proto中定义的message的字段名
- 导出为proto对应的json格式(也可以扩展为导出proto序列化后的二进制数据,以便于更高效的加载)

项目导入
- 加载导出的json数据(或二进制数据),直接反序列化成proto的message对象
