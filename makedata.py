import struct
import numpy as np
import time
import bluefile

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
                data[yy].append(abs(yy))
                if complex_file: #If data is complex, create another entry of the same value. 
                    data[yy].append((yy))

        else:
            for xx in range(x):
                data[yy].append((xx-yy)%127)
                if complex_file: #If data is complex, create another entry of the same value. 
                    data[yy].append((xx-yy))
    return data 

def make_2d_data_np(x,y,file_format):
    """ makes fake 2D data where the data set returned is x by y in size and the value of each value is x-y """

    data = []
    for yy in range(y):
        data.append(np.empty(x))
        if (yy < 1000) :
            print "Making 0 line"
            for xx in range(x):
                data[yy][xx]=(0)
        elif (yy > 5000) :
            print "Making 10 line"
            for xx in range(x):
                data[yy][xx]=(10)
        else:
            for xx in range(x):
                data[yy][xx]=(xx/600)

    return data 


def make_midas_header(xfile,yfile,file_format):
    hdr = bluefile.header(type=2000, format=file_format,subsize=xfile)
    #hdr, data = bluefile.read('mydata_SL_500_1000.tmp')
    hdr['xstart']=0
    hdr['ystart']=0
    hdr['xdelta']=1
    hdr['ydelta']=1
    #hdr['subsize']=xfile
    #hdr['size']=yfile
   # hdr['Version']="BLUE"
   # hdr['head_rep']="EEEI"
   # hdr['data_rep']="EEEI"
    return hdr

if __name__ == "__main__":
    xfile = 6000
    yfile = 6000
    file_format = "SB"
    
    filename = "mydata_%s_%s_%s" %(file_format,xfile,yfile)
    blue = True
    if blue:
        bluefile.set_type2000_format(np.ndarray)
        filename = filename+".tmp"
        hdr = make_midas_header(xfile,yfile,file_format)
        data = make_2d_data_np(xfile,yfile,file_format)
       # print("len data", len(data), len(data[0]),type(data),type(data[0]))
        bluefile.write(filename, hdr, data)

    else:
        data = make_2d_data(xfile,yfile,file_format)
        f= open(filename,"a+")

        for datalist in data:
        # time.sleep(0.1)
            binarydata = pack_list(datalist,file_format)
            f.write(binarydata)

        f.close()

