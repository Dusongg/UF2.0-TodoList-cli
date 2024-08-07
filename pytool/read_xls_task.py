import json
import sys
import xlrd

def extract_data(file_path):
    try:
        workbook = xlrd.open_workbook(file_path)
        sheet1 = workbook.sheet_by_index(2)
        columns1 = [1, 2, 3]
        start_row = 2
        data1 = []
        for row_index in range(start_row, sheet1.nrows):
            row = sheet1.row(row_index)
            row_data = [row[col_index] if col_index < len(row) else '' for col_index in columns1]
            row_data = [cell.value if cell.ctype == xlrd.XL_CELL_TEXT else str(cell.value) for cell in row_data]
            data1.append(row_data)

        sheet1 = workbook.sheet_by_index(3)
        columns2 = [2, 3, 8]
        data2 = []
        for row_index in range(start_row, sheet1.nrows):
            row = sheet1.row(row_index)
            row_data = [row[col_index] if col_index < len(row) else '' for col_index in columns2]
            row_data = [cell.value if cell.ctype == xlrd.XL_CELL_TEXT else str(cell.value) for cell in row_data]
            data2.append(row_data)

        result = {
            'sheet3': data1,
            'sheet4': data2
        }

        return result

    except Exception as e:
        # 捕获异常并将错误信息输出到标准错误
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)  # 使用非零退出码表示错误


if __name__ == "__main__":
    # 从命令行参数获取文件路径
    # if len(sys.argv) != 2:
    #     print("Usage: python extract_data.py <file_path>", file=sys.stderr)
    #     sys.exit(1)
    #
    # file_path = sys.argv[1]
    data = extract_data("D:\\Golang\\OrderManager-cli\\xlsFile\\规范化导出文件.xls")

    # 输出 JSON 格式的数据
    print(json.dumps(data, indent=2))
