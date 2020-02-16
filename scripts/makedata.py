#!/usr/bin/env python
from __future__ import division, print_function
import struct
import numpy as np
import time

import bluefile


def pack_list(data_list, file_format):
    format_map = {
        "F": "f",
        "f": "f",
        "I": "i",
        "i": "i",
        "D": "d",
        "d": "d",
        "L": "i",
        "l": "i",
        "B": "b",
        "b": "b",
    }
    return struct.pack(
        format_map[file_format[1]] * len(datalist),
        *datalist
    )



def make_2d_data(x, y, file_format):
    """ makes fake 2D data where the data set returned is x by y in size and the value of each value is x-y """
    complex_file = False
    if file_format[0] in ("C", "c"):
        complex_file = True

    data = []
    for yy in range(y):
        data.append([])
        if (yy % 25) == 0:
            for xx in range(x):
                data[yy].append(abs(yy))
                if (
                    complex_file
                ):  # If data is complex, create another entry of the same value.
                    data[yy].append((yy))

        else:
            for xx in range(x):
                data[yy].append((xx - yy) % 127)
                if (
                    complex_file
                ):  # If data is complex, create another entry of the same value.
                    data[yy].append((xx - yy))
    return data


def make_2d_data_np(num_columns, num_rows):
    """Makes fake 2D data
    
    The data set returned is x by y in size and the value of each value is x-y
    
    :param x: number of columns
    :param y: number of rows
    :param file_format: format of output file
    :return: numpy array
    """
    data = np.zeros((num_rows, num_columns))
    data[:1000, :] = 0
    data[1000:5000, :] = np.mgrid[:num_columns, :num_columns][1] / 600
    data[5000:, :] = 10
    return data


def make_midas_header(num_columns, num_rows, file_format):
    hdr = bluefile.header(type=2000, format=file_format, subsize=num_columns)
    hdr["xstart"] = 0
    hdr["ystart"] = 0
    hdr["xdelta"] = 1
    hdr["ydelta"] = 1
    return hdr


if __name__ == "__main__":
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument("-r", "--num-rows", type=int, default=6000)
    parser.add_argument("-c", "--num-columns", type=int, default=6000)
    parser.add_argument("-f", "--format", default="SB")
    parser.add_argument("-b", "--blue", action="store_true")
    args = parser.parse_args()

    num_rows = args.num_rows
    num_columns = args.num_columns
    file_format = args.format

    filename = "mydata_%s_%s_%s" % (file_format, num_columns, num_rows)
    if args.blue:
        bluefile.set_type2000_format(np.ndarray)
        filename = filename + ".tmp"
        hdr = make_midas_header(num_columns, num_rows, file_format)
        data = make_2d_data_np(num_columns, num_rows)
        bluefile.write(filename, hdr, data)
    else:
        data = make_2d_data(num_columns, num_rows, file_format)
        with open(filename, "a+") as f:
            for datalist in data:
                binarydata = pack_list(datalist, file_format)
                f.write(binarydata)

