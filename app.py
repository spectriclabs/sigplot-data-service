from flask import Flask
from flask import request
import struct
import numpy

app = Flask(__name__)

def openbinaryfile(filename):
    """ Simple function for opening a binary file. The file name is assumed to have a basename then two number seperated by underscores.
        For example a file name of mydata_1000_2000 will parse to have a x size (framesize) of 1000 and a y value (number of frames) of 2000.
        Need to add a way to get the data element size. At the moment this is hardcoded to 4 bytes"""
    data_element_size = 4
    f = open(filename,"rb")
    (basename,filexsize,fileysize) = str.split(str(filename),"_")
    filexsize = int(filexsize)
    fileysize= int(fileysize)
    return (filexsize,fileysize,data_element_size,f)

def slice_data_from_file(filename,x1,y1,x2,y2):
    """ Returns a 2D slice from a 2D file. Assumes that opening the file, will provide necessary meta data about the framesize and bytes per element. 
        A data points that represent the rectangle of data returned are specified in data elements.
        The returned data will always start with the lowest index point in the return file even if the first point isn't the lowest index point. 
        The return value is a list of numbers"""

    # Opens up a file and get a reference to the file, the 2D file size and bytes per element are returned. This can later be replaced with reading other files as long as the the necessary metadat is also returned.    
    filexsize,fileysize,data_element_size,f = openbinaryfile(filename)
    data = ""

    ystart = min(y1,y2)
    xstart = min(x1,x2)
    ysize = abs(y1-y2)
    xsize = abs(x1-x2)

    # For each needed line to match the ystart and ysize read xsize elements.
    for line in range(ystart,ystart+ysize):
        # Got to start element
        seekvalue = (line*filexsize+xstart)*data_element_size
        f.seek(seekvalue)
        #Read xsize elements
        byte_data = f.read(xsize*data_element_size)
        data+=byte_data

    data = struct.unpack('i'*(xsize*ysize),data)
    return data

def down_sample_data(datain,framesize,outxsize,outysize,transform):
    """ Takes a 2D input dataset (datain list with framesize elements per line) and return a resized data of outxsize, outysize. 
        Uses the specified transform to perform the resizing """
    
    #Check that input data has a whole number of framesize frames
    if (len(datain)/framesize) % 1 != 0:
        print("down sample Data needs whole frames")
        return
    
    inputysize = len(datain)/framesize

    #Check that the data is not be enlarged. Currently don't support any upsacaling.
    if outysize>inputysize or outxsize>framesize:
        print("Current don't support enlarging data sets")
        return

    thinxdata = []
    outdata = []
    xelementsperoutput = framesize/outxsize
    yelementsperoutput = inputysize/outysize


    # First Thin in x Direction. Creates a 2D array that is outxisize wide but still inputysize long
    for y in range(inputysize):
        for x in range(outxsize):
            
            #For each section of transform, find the start and end element
            startelement = y*framesize+int(round(x*xelementsperoutput))
            if x!=(outxsize-1):
                endelement = startelement+ int(numpy.ceil(xelementsperoutput))+1
            else:
                endelement = ((y+1)*framesize)# for last point in output, last point cannot go beyond input size
            
            
            if transform == "mean":
                thinxdata.append(numpy.mean(datain[startelement:endelement]))
            elif transform == "max" : 
                thinxdata.append(max(datain[startelement:endelement]))
            elif transform == "absmax" : 
                thinxdata.append(max(numpy.absolute(datain[startelement:endelement])))
            else:
                print("Transform %s not supported" %(transform))
                return

    # Thin in y Direction
    for y in range(outysize):
        for x in range(outxsize):
            #For each section of transform, find the start, end, and stridesize 
            startelement = x+outxsize*int(round(y*yelementsperoutput))
            if y !=(outysize-1):
                endelement = startelement + (int(numpy.ceil(yelementsperoutput))-1)*outxsize+1
            else:
                endelement = (inputysize-1)*outxsize+x+1
            stridesize = outxsize
            if transform == "mean":
                outdata.append(int(numpy.mean(thinxdata[startelement:endelement:stridesize]))) #currently hardcoded to output int
            elif transform == "max" : 
                outdata.append(max(thinxdata[startelement:endelement:stridesize]))
            elif transform == "absmax" : 
                outdata.append(max(numpy.absolute(thinxdata[startelement:endelement:stridesize])))
            else:
                print("Transform %s not supported" %(transform))
                return
    return outdata

@app.route('/sds')
def split_data():
    filename = request.args.get('filename')
    x1 = int(request.args.get('x1'))
    y1 = int(request.args.get('y1'))
    x2 = int(request.args.get('x2'))
    y2 = int(request.args.get('y2'))
    outxsize = int(request.args.get('outxsize'))
    outysize = int(request.args.get('outysize'))
    transform = str(request.args.get('transform'))

    # Check that requested size is within file

    slicedata = slice_data_from_file(filename,x1,y1,x2,y2)
    framesize = abs(x1-x2)

    down_data = down_sample_data(slicedata,framesize,outxsize,outysize,transform)

    returndata = struct.pack('i'*(outxsize*outysize),*down_data)
    return (returndata)
