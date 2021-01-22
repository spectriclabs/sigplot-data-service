sigplot-data-service
====================

This is a work in progress of a SigPlot data service (sds) that will provide some server-side data thinning and tiling for SigPlot applications. 

#### Support files in repo
* makedata.py - a utility used to make 2D data files for the purpose of testing


## URL

The URL for this service is `<host:port>/sds/<mode>/<ModeSpecificURL>/<LocationName>/path/to/filename`
* `LocationName` needs to match one of the `LocationDetails` structs in the config file
* SDS currently has four possible `mode`s:
    - filesystem (`fs`)
    - file header (`hdr`)
    - Raster Data Service (`rds`)
    - Raster Data Service Tile Mode (`rdstile`)

### File System Mode

* `fs` is filesystem mode. It does not have any mode Specific URL options. This mode interacts with files, directories or location details. 
*  `<host:port>/sds/fs/` - With no location name given it will list the contents of the locaiton configuration
*  `<host:port>/sds/fs/<locationName>/path/to` - with a directory given it will list the contents of that directory 
*  `<host:port>/sds/fs/<locationName>/path/to/filename` - Will return the raw contents of a file on disk

### Hdr Mode

In Header mode (`hdr`) the MIDAS header of a particular file is returned. This is useful to get metadata about the file like it size and type that might inform the parameters for requesting one of the RDS modes. The header is returned as JSON.

The url is `<host:port>/sds/hdr/<ModeSpecificURL>/<LocationName>/path/to/filename`.

### RDS Mode 

RDS Mode is intended to take 2D files and providing two methods to reduce the data size of the files. First a sub-section of the file can be specified so that only data from that subset can be returned instead of the enire file. Second, the selection can thinned or downsampled to create a output that represents the same data but in a smaller represenation.  

First, a subsetion of the file is selected. It is assumed that the data and the selction are 2D. The sub selection is specified by two points (x1,y1) and (x2,y2) that represent the oposite points of a rectangle of the data selction.  

Second, the data slice is passed into a transform where the data is downsized to be of size (`outxsize` by `outysize`). This method supports several different transforms types: mean, max, min, first, and max of absolute value. 

The url for RDS mode is `<host:port>/sds/rds/x1/y1/x2/y2/outxsize/outysize?<optional query paramers>`. The RDS-specific fields are:
* `x1` - x point for the first point of the selection rectangle. 
* `y1` - y point for the first point of the selection rectangle. 
* `x2` - x point for the second point of the selection rectangle. 
* `y2` - y point for the second point of the selection rectangle. 
* `outxsize` - x size of the data output 
* `outysize` - y size of the data output 

Optional Query Parameters:
* `transform` - transform to use to down sample data. Possible options are "max", "min", "mean", "first", "absmax". Default is "first".
* `cxmode` -  Options are "mag", "phase", "real", "imag", "10log", "20log". Default is "mag".
* `outfmt` -  Used to change the output format from what the input file was. Options are "SB", "SI", "SL", "SF", "SD", "RGBA". Type conversion support is limited, does not scale data, trucates decimal. In the case of "RGBA" the value is converted to a RGB value using the colormap and an alpha of 255. Default mode is RGBA.
* `colormap` - Color map names. Currently support "Greyscale", "RampColormap", "ColorWheel", "Spectrum". Default is "RampColormap".
* `zmin` - Value used for RGB mode and sets the minimum value for the color map. If not given the service will find the min and max values from the file and use those values. If the file is larger than 32000 bytes then it will estimate the max and min value based on the first line, the second line, and evenly spaced lines through the middle of the file. 
* `zmax` - Value used for RGB mode and sets the maximum value for the color map. Defaults as describe for zmin.
* `subsize` - x file size or subsize can be given. This can be used for type 1000 files to interupt them as 2D or to override the subsize that is in a type 2000 file. Default is to use the subsize from the file header. 
  
### RDS Tile Mode

RDS Tile mode performs a similar operation to RDS mode, but instead of selecting a custom sized region and a custom size output, the input file is reduced by decimation values specified for x and y and then the outut is broken up into tiles of (`tileXsize` by `tileYSize`) and a single tile is return for each url call.

The url for RDS Tile mode is `<host:port>/sds/rdstile/<tileXsize>/<tileYsize>/<decimationXMode>/<decimationYMode>/<tileX>/<tileY>?<optional query paramers>`
* `tileXsize` - The output tile size in the x direction. Specified in number of points/elements. Possible values are 100,200,300,400,500
* `tileYsize` - The output tile size in the y direction. Specified in number of points/elements. Possible values are 100,200,300,400,500
* `decimationXMode` - Decimation mode for x direction. Modes are values 1-10 the represent power of two decimation (1,2,4,8,...512)
* `decimationYMode` - Decimation mode for y direction. Modes are values 1-10 the represent power of two decimation (1,2,4,8,...512)
* Tile Number in x direction starting at 0 up to the number of tiles in that direction 
* Tile Number in y direction starting at 0 up to the number of tiles in that direction
* The same optional query params are available as described in RDS mode. 

RDS Tiles mode works by thinning the file based on the decimation values provided. If an input file was 3000 by 3000 and a decimation mode for x and y was 3 (deciamte by 4) then the resulting data would be a 750 by 750 file. The those points would be broken up into section based on the tile size. For a tile X size of 100 and a tileYsize of 200, then you would get 8 tiles in each row, the first 7 would have 100 points and the last 50 points. Then 4 tiles in each column with 200 points for the first three, then 150 for the last one. The valid tiles numbesr for x would be 0-7 and y would be 0-3. Tile 7,3 would be the smallest at 50 by 150. 

## Unit Tests
A series of unit tests are available in `sigplot_data_service_test.go`. To run just type `go test` from the source directory. The unit tests use a few data files are are located in th `/tests/` directory. 

## Building

Uses go1.13

## UI Development Mode

```
cd ui
nvm use # assumes you have run nvm install at least once
yarn install
SDS_URL="http://localhost:5055/sds" ROOT_URL="/ui/" ./node_modules/ember-cli/bin/ember serve
```

Now you can visit http://localhost:4200/ui/demo.

## Docker

The Docker version currently *MUST* be run behind an NGINX proxy rooted at /sigplot/

```
make docker

docker run -it --rm -p 5055:5055 sds:0.1
```
