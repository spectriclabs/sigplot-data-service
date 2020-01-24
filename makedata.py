import struct
#import numpy

def pack_list(data_list,file_format):
    
    if file_format[1] in ("F","f"):
        return struct.pack('f' * len(datalist),*datalist)
    elif file_format[1] in ("I","i"):  
        return struct.pack('h' * len(datalist),*datalist)
    elif file_format[1] in ("D","d"):  
        return struct.pack('d' * len(datalist),*datalist)
    elif file_format[1] in ("L","l"):  
        return struct.pack('i' * len(datalist),*datalist)
    elif file_format[1] in ("B","b"):  
        return struct.pack('b' * len(datalist),*datalist)

def make_2d_data(x,y,file_format):
    """ makes fake 2D data where the data set returned is x by y in size and the value of each value is x-y """
    complex_file = False
    if file_format[0] in ("C","c"):
        complex_file = True

    data = []
    for yy in range(y):
        data.append([])
        if (yy % 25) == 0:
            for xx in range(x):
                data[yy].append(yy)
                if complex_file: #If data is complex, create another entry of the same value. 
                    data[yy].append(yy)
        else:
            for xx in range(x):
                data[yy].append(xx-yy)
                if complex_file: #If data is complex, create another entry of the same value. 
                    data[yy].append(xx-yy)
    return data 

if __name__ == "__main__":
    xfile = 8192
    yfile = 20000
    file_format = "SI"
    data = make_2d_data(xfile,yfile,file_format)
    f= open("mydata_%s_%s_%s" %(file_format,xfile,yfile),"w+")
    for datalist in data:
        binarydata = pack_list(datalist,file_format)
        f.write(binarydata)

    f.close()

