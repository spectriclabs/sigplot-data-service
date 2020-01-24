from flask import Flask
from flask import request
import struct
import numpy
import colorcet 
import datetime
import numba
from numba import jit

bluefileAvailable = True
try:
    import bluefile
except ImportError:
    print("Bluefile could not be found")
    bluefileAvailable = False


app = Flask(__name__)

class data_file():

    
    """ Constructor takes byte array and metadata details and constructs data_list """
    def __init__(self,file_ptr,file_format,framesize=1,dataoffset=0):
       
        self.file_ptr = file_ptr #File_ptr to start of data. If file has header, then set this to poing after header
        self.framesize =1 # framesize for 2D data, default to 1 for 1D data
        self.atom_size = 4 #Number of bytes of underlying atom as received or sent
        self.data_element_size = 4 #Number of bytes of underlying data for each element. Twice atom_size for complex
        self.complex = False # True for Complex Data. False for Scaler. Complex data assumed to have two basic elements for each element
        self.struct_string = 'f' # String for using python struct command to pack and unpack the data 
        self.file_format = file_format[1] # FileFormat Character
        
        self.framesize = int(framesize)
        self.dataoffset = dataoffset
        if file_format[0] in ("C","c"):
            self.complex = True
        if file_format[1] in ("F","f"):
            self.atom_size = 4
            self.struct_string = 'f'
        elif file_format[1] in ("I","i"):  
            self.atom_size = 2
            self.struct_string = 'h'
        elif file_format[1] in ("D","d"):  
            self.atom_size = 8
            self.struct_string = "d"
        elif file_format[1] in ("L","l"):  
            self.atom_size = 4
            self.struct_string = 'i'
        elif file_format[1] in ("B","b"):  
            self.atom_size = 1
            self.struct_string = 'h'
        
        if self.complex==True:
            self.data_element_size = 2*self.atom_size
        else:
            self.data_element_size =self.atom_size

    """ Returns data from file starting at first_element for length elements """
    def get_data(self,first_element,length):
        # Got to start element
        self.file_ptr.seek(first_element*self.data_element_size+self.dataoffset)
        #Read  elements
        byte_data = self.file_ptr.read(length*self.data_element_size)
        if self.complex:
            length = length*2
        data= np.array(struct.unpack(self.struct_string*length,byte_data))
        return data

@jit()
def get_bytes_from_file(file_name,first_element,length,data_element_size,dataoffset,struct_char,data_complex):
        file_ptr = open(file_name,"rb")
        file_ptr.seek(first_element*data_element_size+dataoffset)
        #Read  elements
        byte_data = file_ptr.read(length*data_element_size)
        if data_complex:
            length = length*2
        data= numpy.array(struct.unpack(struct_char*length,byte_data))
        return data

transform_code = {
    "mean" :1,
    "max" :2,
    "min" :3,
    "absmax":4,
    "first":5
}

def getstructstring(file_format):

        if file_format in ("F","f","SF","sf"):
            return "f"
        elif file_format in ("I","i","SI", "si"):  
            return "h"
        elif file_format in ("D","d","SD","sd"):  
            return"d"
        elif file_format in ("L","l","SL","sl"):  
            return "i"
        elif file_format in ("B","b","SB","sb"):  
            return "h"
        else:
            return None


def openbluefile(filename):
    header,f  = bluefile.readheader(filename,keepopen=True)
    file_format = header['format']
    framesize = header['subsize']
    file_data = data_file(f,file_format,framesize=framesize,dataoffset = int(header['data_start']))
    return file_data

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



# def slice_data_from_file(file_data,x1,y1,x2,y2):
#     """ Returns a 2D slice from a 2D file. Assumes that opening the file, will provide necessary meta data about the framesize and bytes per element. 
#         A data points that represent the rectangle of data returned are specified in data elements.
#         The returned data will always start with the lowest index point in the return file even if the first point isn't the lowest index point. 
#         The return value is a list of numbers"""


#     data = []

#     ystart = min(y1,y2)
#     xstart = min(x1,x2)
#     ysize = abs(y1-y2)
#     xsize = abs(x1-x2)
    
#     # For each needed line to match the ystart and ysize read xsize elements.
#     for line in range(ystart,ystart+ysize):
#         start = (line*file_data.framesize+xstart)
#         mydata =file_data.get_data(start,xsize)
#         data+=mydata
#     return data


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

def applycolormap(datain,colormap,zmin,zmax):
    dataout = []
    color_per_span = (zmax-zmin)/256.0
    for data in datain:
        color_value = int(min(max((data-zmin)/color_per_span,0),255))
        try:
            color_string = (colorcet.palette[colormap][color_value])[1:]
        except KeyError:
            raise Exception("Invalid ColorMap")
        dataout+=[int(color_string[i:i+2], 16) for i in (0, 2, 4)]
    return dataout

@jit(nopython=True)
def down_sample_data_inx(datain,framesize,outxsize,transform_code):
    inputysize = int(len(datain)/framesize)
    xelementsperoutput = float(framesize/outxsize)
    thinxdata_array =numpy.empty(int(inputysize*outxsize))

    for y in range(inputysize):
        for x in range(outxsize):
            
            #For each section of transform, find the start and end element
            startelement = y*framesize+int(round(x*xelementsperoutput))
            if x!=(outxsize-1):
                endelement = startelement+ int(numpy.ceil(xelementsperoutput))
            else:
                endelement = ((y+1)*framesize)# for last point in output, last point cannot go beyond input size
                startelement = endelement - int(numpy.ceil(xelementsperoutput))

            if transform_code ==1: #mean
                thinxdata_array[y*outxsize+x] = numpy.mean(datain[startelement:endelement])
            elif transform_code ==2: #max
                thinxdata_array[y*outxsize+x] = numpy.max(datain[startelement:endelement])
            elif transform_code ==3: #min
                thinxdata_array[y*outxsize+x] = numpy.min(datain[startelement:endelement])
            elif transform_code ==4: #Max abs 
                thinxdata_array[y*outxsize+x] = numpy.max(numpy.absolute(datain[startelement:endelement]))
            elif transform_code ==5: #first
                thinxdata_array[y*outxsize+x] = datain[startelement]

    return thinxdata_array

# def down_sample_data_inx(datain,framesize,outxsize,transform_code):
#     """ Takes a 2D input dataset (datain list with framesize elements per line) and return a resized data of outxsize, outysize. 
#         Uses the specified transform to perform the resizing """
    
#     #Check that input data has a whole number of framesize frames
#     if (len(datain)/framesize) % 1 != 0:
#         raise Exception("down sample Data needs whole frames")
    
#     inputysize = len(datain)/framesize

#     #Check that the data is not be enlarged. Currently don't support any upsacaling.
#     if outxsize>framesize:
#         raise Exception("Current don't support enlarging data sets")


#     # First Thin in x Direction. Creates a 2D array that is outxisize wide but still inputysize long
#     thinxdata_array = numpy.array([])
#     thinxdata_array = down_sample_data_inx(datain_array,framesize,outxsize,transform_code)

#     return thinxdata


"""     #Mean has a different (faster) implemenation
    elif transform=="mean2":
        for y in range(inputysize):
            for x in range(outxsize):
                
                #For each section of transform, find the start and end element
                startelement = y*framesize+int(round(x*xelementsperoutput))
                if x!=(outxsize-1):
                    endelement = startelement+ int(numpy.ceil(xelementsperoutput))
                else:
                    endelement = ((y+1)*framesize)# for last point in output, last point cannot go beyond input size
                    startelement = endelement - int(numpy.ceil(xelementsperoutput))
                thinxdata.append(datain[startelement:endelement])
                #numpy.append(thinxdata_array,datain[startelement:endelement])
        thinxdata_array_mean=numpy.mean(thinxdata,axis=1)
        thinxdata = thinxdata_array_mean.tolist()

    else:
        for y in range(inputysize):
            for x in range(outxsize):
                
                #For each section of transform, find the start and end element
                startelement = y*framesize+int(round(x*xelementsperoutput))
                if x!=(outxsize-1):
                    endelement = startelement+ int(numpy.ceil(xelementsperoutput))
                else:
                    endelement = ((y+1)*framesize)# for last point in output, last point cannot go beyond input size
                    startelement = endelement - int(numpy.ceil(xelementsperoutput))
                
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
                    return """




def down_sample_data_iny(thinxdata,outxsize,outysize,transform):
    #print("down_sample_data_iny", len(thinxdata),type(thinxdata),outxsize,outysize,transform)
    inputysize = len(thinxdata)/outxsize
    outdata = []
    yelementsperoutput = inputysize/outysize
    
    #Check that the data is not be enlarged. Currently don't support any upsacaling.
    if outysize>inputysize:
        print("Current don't support enlarging data sets")
        return

    # Thin in y Direction
    if transform == "mean2":
        for y in range(outysize):
            for x in range(outxsize):
                #For each section of transform, find the start, end, and stridesize 
                startelement = x+outxsize*int(round(y*yelementsperoutput))
                if y !=(outysize-1):
                    endelement = startelement + (int(numpy.ceil(yelementsperoutput))-1)*outxsize+1
                else:
                    endelement = (inputysize-1)*outxsize+x+1
                    startelement = endelement - (int(numpy.ceil(yelementsperoutput))-1)*outxsize-1
                stridesize = outxsize
                outdata.append(thinxdata[startelement:endelement:stridesize])
        outdata = numpy.mean(outdata,axis=1)
    else:
        for y in range(outysize):
            for x in range(outxsize):
                #For each section of transform, find the start, end, and stridesize 
                startelement = x+outxsize*int(round(y*yelementsperoutput))
                if y !=(outysize-1):
                    endelement = startelement + (int(numpy.ceil(yelementsperoutput))-1)*outxsize+1
                else:
                    endelement = (inputysize-1)*outxsize+x+1
                    startelement = endelement - (int(numpy.ceil(yelementsperoutput))-1)*outxsize-1
                stridesize = outxsize
                if transform == "mean":
                    outdata.append(numpy.mean(thinxdata[startelement:endelement:stridesize])) 
                elif transform == "mean_numba":
                    outdata.append(numpy.mean(thinxdata[startelement:endelement:stridesize])) 
                elif transform == "max" : 
                    outdata.append(max(thinxdata[startelement:endelement:stridesize]))
                elif transform == "absmax" : 
                    outdata.append(max(numpy.absolute(thinxdata[startelement:endelement:stridesize])))
                elif transform == "min" : 
                    outdata.append(min(thinxdata[startelement:endelement]))
                elif transform == "first" : 
                    outdata.append(thinxdata[startelement])
                else:
                    print("Transform %s not supported" %(transform))
                    return
    return outdata

# @jit(nopython=True)
# def processdata_inx(datain_array,xstart,ystart,xsize,ysize,cxmode,outxsize,transform_code):
#     down_data_x = numpy.array.empty(int(inputysize*outxsize))


#         #if file_data.complex:
#         #    cx_slicedata = apply_cxmode(slicedata,cxmode)
#         #else:
#         #    cx_slicedata = slicedata
        
#         #datain_array = numpy.array(cx_slicedata)
#         down_data_x = down_sample_data_inx(datain_array[line],xsize,outxsize,transform_code)
#         down_data_x += thinxdata_array.tolist()
#         #down_data_x = down_sample_data_inx(cx_slicedata,xsize,outxsize,transform)
#     return down_data_x

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
    zmin = (request.args.get('zmin'))
    zmax = (request.args.get('zmax'))
    
    # Apply Default to optional Parameters
    if cxmode =='None':
        cxmode="mag"
    

    
    if colormap =='None':
        colormap="rainbow"


    # TODO - Later add support for more file types and implement methods that parse the metadata and return data_file objects
    if ".tmp" in filename or ".prm" in filename:
        file_type = "blue"
    else:
        file_type = "binary_underscore"

    if file_type == "binary_underscore":
        file_data = openbinaryfile(filename)
    elif file_type == "blue":
        if not(bluefileAvailable):
            raise Exception("Bluefile Support is not available")
        file_data = openbluefile(filename)
    else:
        raise Exception("Unsupported file type")

    if outfmt =='None':
        outfmt =file_data.file_format



    ystart = min(y1,y2)
    xstart = min(x1,x2)
    ysize = abs(y1-y2)
    xsize = abs(x1-x2)
    down_data_x = []
    filedata = numpy.empty((ysize,xsize))
    # Step 1. Read Data From File
    print("1. ",datetime.datetime.now())
    for line in range(ystart,ystart+ysize):
        start = (line*xsize+xstart)
        filedata[line]=get_bytes_from_file(filename,start,xsize,file_data.data_element_size,file_data.dataoffset,file_data.struct_string,file_data.complex)
        #slicedata =file_data.get_data(start,)

    # Step 2. For each line of input that needs to be processed, read the line, apply cxmode, and downsize to output size
    print("2. ",datetime.datetime.now())
    down_data_array = down_sample_data_inx(filedata,xsize,outxsize,transform_code[transform])
    #down_data_array = processdata_inx(filedata,xstart,ystart,xsize,ysize,cxmode,outxsize,transform_code[transform])
    down_data_x = down_data_array.to_list()
    #for line in range(ystart,ystart+ysize):
    #    down_data_x+=processline(file_data,xstart,line,xsize,cxmode,outxsize,transform)
    
    # Step 3 Take all lines processed and down sample data in y dimention to fit outsize
    print("3. ",datetime.datetime.now())
    down_data = down_sample_data_iny(down_data_x,outxsize,outysize,transform)
    
    # Step 4 apply output formatting
    print("4. ",datetime.datetime.now())
    if outfmt in ("RGB", "rgb"):
        if zmin==None:
            zmin= int(min(cx_slicedata))
        else:
            zmin = int(zmin)
        if zmax==None:
            zmax= int(max(cx_slicedata))
        else:
            zmax = int(zmax)
        returndata = applycolormap(down_data,colormap,zmin,zmax)
        returndata = struct.pack('B'*(outxsize*outysize*3),*returndata)
    else:
        out_file_struct_string = getstructstring(outfmt)
        if not(out_file_struct_string):
            raise Exception("Invalid outfmt mode: %s" % (cxmode))
        if out_file_struct_string in ('h','i','l'):
            down_data = [int(x) for x in down_data]
        returndata = struct.pack(out_file_struct_string*(outxsize*outysize),*down_data)
    print("4. ",datetime.datetime.now())
    
    # Make Output Response and put metadata in response headers
    resp = app.make_response(returndata)
    resp.headers['outfmt'] = outfmt
    resp.headers['framesize'] = outxsize
    resp.headers['colormap'] = colormap
    resp.headers['zmin'] = zmin
    resp.headers['zmax'] = zmax

    return (resp)


