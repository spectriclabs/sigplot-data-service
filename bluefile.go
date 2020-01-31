package main

type BlueHeader struct {
	Version 	[4]byte 	//Header Version
	Head_rep 	[4]byte		//Header representation
	Data_rep 	[4]byte		//Data representation
	Detached 	int32		//Detached Header
	Protected 	int32		//Protected from overwrite
	Pipe      	int32		//Pipe mode (N/A)
	Ext_start 	int32       //Extended header start, in 512-byte blocks
	Ext_size 	int32       //Extended header size in bytes
	Data_start 	float64     //Data start in bytes
	Data_size   float64     //Data size in bytes
	File_type	int32       //File type code
	Format 		[2]byte     //Data format code
	Flagmask	int16       //16-bit flagmask (1=flagbit)
	Timecode	float64     //Time code field
	Inlet		int16       //Inlet owner
	Outlets		int16       //Number of outlets
	Outmask		int32       //Outlet async mask
	Pipeloc		int32       //Pipe location
	Pipesize	int32       //Pipe size in bytes
	In_byte		float64     //Next input byte
	Out_byte	float64     //Next out byte (cumulative)
	Outbytes	[8]float64  //Next out byte (each outlet)
	Keylength	int32       //Length of keyword string
	Keywords	[92]byte    //User defined keyword string
	Xstart 		float64     //Frame (column) starting value
	Xdelta  	float64     //Increment between samples in frame
	Xunits 		int32       //Frame (column) units
	Subsize 	int32       //Number of data points per frame (row)
	Ystart 		float64     //Abscissa (row) start
	Ydelta 		float64     //Increment between frames
	Yunits		int32       //Abscissa (row) unit code
	Adjunct     [216]byte   //Type-specific adjunct union (See bel
}

type BlueHeaderShortenedFields struct {
	Version 	string 		`json:"version"`	//Header Version
	Head_rep 	string		`json:"head_rep"`	//Header representation
	Data_rep 	string		`json:"data_rep"`	//Data representation
	Detached 	int32		`json:"detached"`	//Detached Header
	Protected 	int32		`json:"protected"`	//Protected from overwrite
	Pipe      	int32		`json:"pipe"`		//Pipe mode (N/A)
	Ext_start 	int32       `json:"ext_start"`	//Extended header start, in 512-byte blocks
	Ext_size 	int32       `json:"ext_size"`	//Extended header size in bytes
	Data_start 	float64     `json:"data_start"`	//Data start in bytes
	Data_size   float64     `json:"data_size"`	//Data size in bytes
	File_type	int32       `json:"file_type"`	//File type code
	Format 		string      `json:"format"`		//Data format code
	Flagmask	int16       `json:"flagmask"`	//16-bit flagmask (1=flagbit)
	Timecode	float64     `json:"timecode"`	//Time code field
	Xstart 		float64     `json:"xstart"` 	//Frame (column) starting value
	Xdelta  	float64     `json:"xdelta"`		//Increment between samples in frame
	Xunits 		int32       `json:"xunits"`		//Frame (column) units
	Subsize 	int32       `json:"subsize"`	//Number of data points per frame (row)
	Ystart 		float64     `json:"ystart"`		//Abscissa (row) start
	Ydelta 		float64     `json:"ydelta"`		//Increment between frames
	Yunits		int32       `json:"yunits"`		//Abscissa (row) unit code
	Spa			int     	`json:"spa"`		//scalars per atom
	Bps 		float64     `json:"bps"`		// bytes per scalar
	Bpa			float64    	`json:"bpa"`		//bytes per atom
	Ape			int       	`json:"ape"`		//atoms per element
	Bpe			float64    	`json:"bpe"`		//bytes per element
	Size		int       	`json:"size"`	 	//number of elements in dview
}
