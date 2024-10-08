# read_xls.py
import xlrd
import json
import sys


def read_xls(file_path):
    try:
        workbook = xlrd.open_workbook(file_path)
        sheet = workbook.sheet_by_index(0)

        columns = [0, 1, 2, 3, 12, 14, 19, 48]
        start_row = 2

        data = []

        for row_index in range(start_row, sheet.nrows):
            row = sheet.row(row_index)
            row_data = [row[col_index] if col_index < len(row) else '' for col_index in columns]
            row_data = [cell.value if cell.ctype == xlrd.XL_CELL_TEXT else str(cell.value) for cell in row_data]
            data.append(row_data)

        return data

    except Exception as e:
        # 捕获异常并将错误信息输出到标准错误
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)  # 使用非零退出码表示错误


if __name__ == '__main__':
    if len(sys.argv) != 2:
        print("Usage: python read_xls.py <file_path>",file=sys.stderr)
        sys.exit(1)

    file_path = sys.argv[1]
    data = read_xls(file_path)

    # Print data as JSON
    print(json.dumps(data))
