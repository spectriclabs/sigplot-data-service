from flask import Flask
from flask import request
import struct
import numpy

app = Flask(__name__)

class data_file():

    
    """ Constructor takes byte array and metadata details and constructs data_list """
    def __init__(self,file_ptr,file_format,framesize=1):
       
        self.file_ptr = file_ptr #File_ptr to start of data. If file has header, then set this to poing after header
        self.framesize =1 # framesize for 2D data, default to 1 for 1D data
        self.atom_size = 4 #Number of bytes of underlying atom as received or sent
        self.data_element_size = 4 #Number of bytes of underlying data for each element. Twice atom_size for complex
        self.complex = False # True for Complex Data. False for Scaler. Complex data assumed to have two basic elements for each element
        self.struct_string = "f" # String for using python struct command to pack and unpack the data 
        
        
        self.framesize = int(framesize)
        if file_format[0] in ("C","c"):
            self.complex = True
        
        if file_format[1] in ("F","f"):
            self.atom_size = 4
            self.struct_string = "f"
        elif file_format[1] in ("I","i"):  
            self.atom_size = 2
            self.struct_string = "h"
        elif file_format[1] in ("D","d"):  
            self.atom_size = 8
            self.struct_string = "d"
        elif file_format[1] in ("L","l"):  
            self.atom_size = 4
            self.struct_string = "i"
        elif file_format[1] in ("B","b"):  
            self.atom_size = 1
            self.struct_string = "h"
        
        if self.complex==True:
            self.data_element_size = 2*self.atom_size

    """ Returns data from file starting at first_element for length elements """
    def get_data(self,first_element,length):
        # Got to start element
        self.file_ptr.seek(first_element*self.data_element_size)
        #Read  elements
        byte_data = self.file_ptr.read(length*self.data_element_size)
        if self.complex:
            length = length*2
        data= struct.unpack(self.struct_string*length,byte_data)
        return data



def openbinaryfile(filename):
    """ Simple function for opening a binary file. The file name is assumed to have a basename,file_format, then two number seperated by underscores.
        For example a file name of mydata_SL_1000_2000 will parse to have file_format of SL, a x size (framesize) of 1000 and a y value (number of frames) of 2000.
        Returns a data_file object, which has a pointer to the file and the metadata in a class"""

    f = open(filename,"rb")
    try:
        (basename,file_format,filexsize,fileysize) = str.split(str(filename),"_")
    except ValueError:
        raise Exception("Invalid binary File Name. Looking for <basename>_<file_forma>_<xsize>_<ysize>")
    
    file_data = data_file(f,file_format,framesize=filexsize)

    return file_data

def slice_data_from_file(file_data,x1,y1,x2,y2):
    """ Returns a 2D slice from a 2D file. Assumes that opening the file, will provide necessary meta data about the framesize and bytes per element. 
        A data points that represent the rectangle of data returned are specified in data elements.
        The returned data will always start with the lowest index point in the return file even if the first point isn't the lowest index point. 
        The return value is a list of numbers"""


    data = []

    ystart = min(y1,y2)
    xstart = min(x1,x2)
    ysize = abs(y1-y2)
    xsize = abs(x1-x2)
    
    # For each needed line to match the ystart and ysize read xsize elements.
    for line in range(ystart,ystart+ysize):
        start = (line*file_data.framesize+xstart)
        mydata =file_data.get_data(start,xsize)
        data+=mydata
    return data


def apply_cxmode(datain,cxmode,lo_thresh=1.0e-20):
    """ Applies cx_mode to datain. Will return a data list that is half as long. cxmode options are 'mag','phase','real','imag','10log','20log'."""
    dataout = []
    for i in range(0,len(datain)-1,2):
        cxpoint = datain[i]+datain[i+1]*1j
        if cxmode=="mag":
            dataout.append(numpy.absolute(cxpoint))
        elif cxmode=="phase":
            dataout.append(numpy.angle(cxpoint))
        elif cxmode=="real":
            dataout.append(numpy.real(cxpoint))
        elif cxmode=="imag":
            dataout.append(numpy.imag(cxpoint))
        elif cxmode=="10log":
            dataout.append(10*numpy.log10(max(datain[i]**2+datain[i+1]**2,lo_thresh))) 
        elif cxmode=="20log":
            dataout.append(20*numpy.log10(max(datain[i]**2+datain[i+1]**2,lo_thresh))) 
        else:
            raise Exception("Invalid cx mode: %s" % (cxmode))

    return dataout

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
                endelement = startelement+ int(numpy.ceil(xelementsperoutput))
            else:
                endelement = ((y+1)*framesize)# for last point in output, last point cannot go beyond input size
            
            if transform == "mean":
                thinxdata.append(numpy.mean(datain[startelement:endelement]))
            elif transform == "max" : 
                thinxdata.append(max(datain[startelement:endelement]))
            elif transform == "absmax" : 
                thinxdata.append(max(numpy.absolute(datain[startelement:endelement])))
            elif transform == "min" : 
                thinxdata.append(min(datain[startelement:endelement]))
            elif transform == "first" : 
                thinxdata.append(datain[startelement])
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
            elif transform == "min" : 
                outdata.append(min(datain[startelement:endelement]))
            elif transform == "first" : 
                outdata.append(datain[startelement])
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
    cxmode = str(request.args.get('cxmode'))
    outfmt = str(request.args.get('outfmt'))
    colormap = str(request.args.get('colormap'))

    # Apply Default to optional Parameters
    if cxmode =='None':
        cxmode="mag"
    
    if outfmt =='None':
        outfmt ="passthrough"
    
    if colormap =='None':
        colormap="rainbow"


    # TODO - Later add support for more file types and implement methods that parse the metadata and return data_file objects
    file_type = "binary_underscore"

    if file_type == "binary_underscore":
        file_data = openbinaryfile(filename)
    else:
        raise Exception("Unsupported file type")

    slicedata = slice_data_from_file(file_data,x1,y1,x2,y2)
    framesize = abs(x1-x2)

    if file_data.complex:
        cx_slicedata = apply_cxmode(slicedata,cxmode)
    else:
        cx_slicedata = slicedata

    down_data = down_sample_data(cx_slicedata,framesize,outxsize,outysize,transform)

    returndata = struct.pack(file_data.struct_string*(outxsize*outysize),*down_data)
    return (returndata)


