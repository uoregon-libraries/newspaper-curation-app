package alto

var altoTemplateString = `
<?xml version="1.0" encoding="utf-8"?>
<alto xmlns="http://schema.ccs-gmbh.com/ALTO">
  <Description>
    <MeasurementUnit>inch1200</MeasurementUnit>
    <sourceImageInformation>
      <fileName>{{.PDFFilename}}</fileName>
    </sourceImageInformation>
    <OCRProcessing ID="OCR.0">
      <ocrProcessingStep>
        <processingStepSettings>N/A</processingStepSettings>
        <processingSoftware>
          <softwareCreator>UO Libraries</softwareCreator>
          <softwareName>Batch Maker</softwareName>
          <softwareVersion>1.0.0</softwareVersion>
        </processingSoftware>
      </ocrProcessingStep>
    </OCRProcessing>
  </Description>
  <Styles>
    <TextStyle ID="TS_10.0" FONTSIZE="10.0" />
  </Styles>
  <Layout>
  <Page ID="PAGE.0" HEIGHT="{{.PageHeight}}" WIDTH="{{.PageWidth}}" PHYSICAL_IMG_NR="{{.ImageNumber}}" PROCESSING="OCR.0" PC="0.99">
    <PrintSpace ID="PS.0" HEIGHT="{{.PageHeight}}.0" WIDTH="{{.PageWidth}}.0" HPOS="0.0" VPOS="0.0">
{{range .Flows -}}
  {{range .Blocks -}}
    {{$blockIndex := NextBlockNumber -}}
    {{$blockid := (printf "TB.%d.%d" $.ImageNumber $blockIndex)}}
      <TextBlock xmlns:ns{{$blockIndex}}="http://www.w3.org/1999/xlink" ID="{{$blockid}}" {{MakeCoordAttrs .Rect}} ns{{$blockIndex}}:type="simple" language="en">
        {{range $index, $line := .Lines -}}
        {{$lineid := (printf "%s_%d" $blockid $index) -}}
        <TextLine ID="{{$lineid}}" {{MakeCoordAttrs .Rect}}>
          {{range $index, $word := .Words -}}
          {{$wordid := (printf "%s_%d" $lineid $index) -}}
            <String ID="{{$wordid}}" STYLEREFS="TS_10.0" {{MakeCoordAttrs .Rect}} CONTENT="{{.Text}}" WC="0.99" />
          {{end}}
        </TextLine>
        {{- end}}
      </TextBlock>
  {{- end}}
{{end}}
    </PrintSpace>
  </Page>
</Layout>
</alto>
`