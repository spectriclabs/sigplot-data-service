# project/test_basic.py
 
 
import os
import unittest
import sys
sys.path.append("..")
from app import app
import struct
import numpy
 
class BasicTests(unittest.TestCase):
 
    ############################
    #### setup and teardown ####
    ############################
 
    # executed prior to each test
    def setUp(self):
        app.config['TESTING'] = True
        app.config['DEBUG'] = False
        self.app = app.test_client()
  
    # executed after each test
    def tearDown(self):
        pass
 
 
###############
#### tests ####
###############
 
    def test_basic(self):
        filename = "../mydata_SL_500_1000"
        x1 = 10 
        y1 = 10
        x2 = 300
        y2 = 300
        outxsize = 100
        outysize = 100
        transform = "max"
        #response = self.app.get('/sds', filename=filename,x1=x1,y1=y1,x2=x2,y2=y2,outxsize=outxsize,outysize=outysize,transform=transform)
        response = self.app.get('/sds?filename=%s&x1=%i&y1=%i&x2=%i&y2=%i&outxsize=%i&outysize=%i&transform=%s' 
                                %(filename,x1,y1,x2,y2,outxsize,outysize,transform))

        self.assertEqual(len(response.data),outxsize*outysize*4)

    def test_whole_file(self):
        filename = "../mydata_SL_500_1000"
        x1 = 0 
        y1 = 0
        x2 = 500
        y2 = 1000
        outxsize = 500
        outysize = 1000
        transform = "max"
        response = self.app.get('/sds?filename=%s&x1=%i&y1=%i&x2=%i&y2=%i&outxsize=%i&outysize=%i&transform=%s' 
                                %(filename,x1,y1,x2,y2,outxsize,outysize,transform))

        self.assertEqual(len(response.data),outxsize*outysize*4)
        data = struct.unpack('i'*(len(response.data)/4),response.data)

        f = open(filename,"rb") 
        filedata = f.read(len(response.data))
        filedata = struct.unpack('i'*(len(response.data)/4),filedata)

        self.assertEqual(filedata[0],data[0])
        self.assertEqual(filedata[-1],data[-1])
        self.assertEqual(filedata[500],data[500])

    def test_single_points(self):
        filename = "../mydata_SL_500_1000"
        framesize =500
        fileysize =1000
        atom_size = 4
        testpoints = ([0,0],[499,0],[1,0],[0,999],[499,999],[100,100])
        transform = "max"
        f = open(filename,"rb") 
        filedata = f.read(atom_size*framesize*fileysize)
        filedata = struct.unpack('i'*(framesize*fileysize),filedata)
        for point in testpoints:
            response = self.app.get('/sds?filename=%s&x1=%i&y1=%i&x2=%i&y2=%i&outxsize=%i&outysize=%i&transform=%s' 
                                %(filename,point[0],point[1],point[0]+1,point[1]+1,1,1,transform))
            data = struct.unpack('i'*(len(response.data)/atom_size),response.data)
            self.assertEqual(filedata[point[1]*framesize+point[0]],data[0])

    def test_slice(self):
        filename = "../mydata_SL_500_1000"
        atom_size = 4
        framesize =500
        fileysize =1000
        x1 = 0 
        y1 = 0
        x2 = 10
        y2 = 10
        outxsize = 5
        outysize = 5
        transform = "mean"
        response = self.app.get('/sds?filename=%s&x1=%i&y1=%i&x2=%i&y2=%i&outxsize=%i&outysize=%i&transform=%s' 
                                %(filename,x1,y1,x2,y2,outxsize,outysize,transform))

        self.assertEqual(len(response.data),outxsize*outysize*atom_size)
        data = struct.unpack('i'*(len(response.data)/atom_size),response.data)

        f = open(filename,"rb") 
        filedata = f.read(atom_size*framesize*fileysize)
        filedata = struct.unpack('i'*(framesize*fileysize),filedata)

        # Spot Check some point in the output slice. We are doing a 2X reduction so all points should be an average of two points from one line and two from the next line
        meandata = (filedata[0],filedata[1],filedata[500],filedata[501])
        self.assertEqual(int(numpy.mean(meandata)),data[0])
        meandata = (filedata[2],filedata[3],filedata[502],filedata[503])
        self.assertEqual(int(numpy.mean(meandata)),data[1])
        meandata = (filedata[8],filedata[9],filedata[508],filedata[509])
        self.assertEqual(int(numpy.mean(meandata)),data[4])
        meandata = (filedata[1004],filedata[1005],filedata[1504],filedata[1505])
        self.assertEqual(int(numpy.mean(meandata)),data[7])

    def test_slice_float_file(self):
        filename = "../mydata_SF_500_1000"
        atom_size = 4
        framesize =500
        fileysize =1000
        x1 = 0 
        y1 = 0
        x2 = 10
        y2 = 10
        outxsize = 5
        outysize = 5
        transform = "mean"
        response = self.app.get('/sds?filename=%s&x1=%i&y1=%i&x2=%i&y2=%i&outxsize=%i&outysize=%i&transform=%s' 
                                %(filename,x1,y1,x2,y2,outxsize,outysize,transform))

        self.assertEqual(len(response.data),outxsize*outysize*atom_size)
        data = struct.unpack('f'*(len(response.data)/atom_size),response.data)

        f = open(filename,"rb") 
        filedata = f.read(atom_size*framesize*fileysize)
        filedata = struct.unpack('f'*(framesize*fileysize),filedata)

        # Spot Check some point in the output slice. We are doing a 2X reduction so all points should be an average of two points from one line and two from the next line
        meandata = (filedata[0],filedata[1],filedata[500],filedata[501])
        self.assertEqual(int(numpy.mean(meandata)),data[0])
        meandata = (filedata[2],filedata[3],filedata[502],filedata[503])
        self.assertEqual(int(numpy.mean(meandata)),data[1])
        meandata = (filedata[8],filedata[9],filedata[508],filedata[509])
        self.assertEqual(int(numpy.mean(meandata)),data[4])
        meandata = (filedata[1004],filedata[1005],filedata[1504],filedata[1505])
        self.assertEqual(int(numpy.mean(meandata)),data[7])

    def test_slice_float_complex_file(self):
        filename = "../mydata_CF_500_1000"
        atom_size = 4
        framesize =500
        fileysize =1000
        x1 = 0 
        y1 = 0
        x2 = 10
        y2 = 10
        outxsize = 5
        outysize = 5
        transform = "mean"
        cxmode = "real"
        response = self.app.get('/sds?filename=%s&x1=%i&y1=%i&x2=%i&y2=%i&outxsize=%i&outysize=%i&transform=%s&cxmode=%s' 
                                %(filename,x1,y1,x2,y2,outxsize,outysize,transform,cxmode))

        self.assertEqual(len(response.data),outxsize*outysize*atom_size)
        data = struct.unpack('f'*(len(response.data)/atom_size),response.data)

        # The test files for complex have duplicate values for i and q. So if we specify the 'real' or 'img' cxmode we should get the same answer as the scale file.
        filename = "../mydata_SF_500_1000"
        f = open(filename,"rb") 
        filedata = f.read(atom_size*framesize*fileysize)
        filedata = struct.unpack('f'*(framesize*fileysize),filedata)

        # Spot Check some point in the output slice. We are doing a 2X reduction so all points should be an average of two points from one line and two from the next line
        meandata = (filedata[0],filedata[1],filedata[500],filedata[501])
        self.assertEqual(int(numpy.mean(meandata)),data[0])
        meandata = (filedata[2],filedata[3],filedata[502],filedata[503])
        self.assertEqual(int(numpy.mean(meandata)),data[1])
        meandata = (filedata[8],filedata[9],filedata[508],filedata[509])
        self.assertEqual(int(numpy.mean(meandata)),data[4])
        meandata = (filedata[1004],filedata[1005],filedata[1504],filedata[1505])
        self.assertEqual(int(numpy.mean(meandata)),data[7])

if __name__ == "__main__":
    unittest.main()

