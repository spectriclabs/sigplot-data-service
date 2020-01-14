import struct
#import numpy

def make_2d_data(x,y):
    """ makes fake 2D data where the data set returned is x by y in size and the value of each value is x-y """
    data = []
    for yy in range(y):
        data.append([])
        if (yy % 25) == 0:
            for xx in range(x):
                data[yy].append(yy)
        else:
            for xx in range(x):
                data[yy].append(xx-yy)
    return data 

if __name__ == "__main__":
    xfile = 500
    yfile = 1000
    data = make_2d_data(xfile,yfile)
    f= open("mydata_%s_%s" %(xfile,yfile),"w+")
    for datalist in data:
        binarydata = struct.pack('i' * len(datalist),*datalist)
        f.write(binarydata)

    f.close()

