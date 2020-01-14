# sigplot-data-service

This is a work in progress of a Sigplot data service (sds) that will provide some serverside data thining for sigplot applications. 

##Files currently in repo
app.py - A flask web app the provides the sds 
makedata.py - a utility used to make 2D data files for the purpose of testing
sds.ipynb - a Juypter notebook for interacting and tests the SDS that can request data and plot it.


## SDS 

The SDS is intended to take 2D files and providing two methods to reduce the data sze of the files. First a sub section of the file can be specified so that only data from that subset can be returned instead of the enire file. Second, the selection can thinned or downsample to create a output that represents the same data but is smaller.  

First slide_data_from_file is called and returns only the subset of the file that was requested. It is assumed that the data and the selction are 2D. The sub selection is specified by two points (x1,y1) and (x2,y2) that represent the oposite points of a rectangle of the data selction.  

Second the data slice is passed into down_sample_data where the data is downsized to be of size (outxsize by outysize). This method supports several different transform types, mean, max, min, first, and absolute max. 

Currently the web service has one end point /sds that takes 8 parameters:
  * filename - path/file name to the file from where the app.py is running. 
  * x1 - x point for the first point of the selection rectangle. 
  * y1 - y point for the first point of the selection rectangle. 
  * x2 - x point for the second point of the selection rectangle. 
  * y2 - y point for the second point of the selection rectangle. 
  * outxsize - x size of the data output 
  * outysize - y size of the data output 
  * transform - transform to use to down sample data. Possible options 'max', 'min', 'mean', 'first', 'absmax'
  
## Juypter testing

The sds.ipynb jupyter notebook can be used to interact with the SDS and plot the 2D files and the sub selections that come back to SDS. It is assumed that Juypter can find the same data files in the same relative path as the server side application. This is only needed to test and compare the files before and after the sub selections. 
