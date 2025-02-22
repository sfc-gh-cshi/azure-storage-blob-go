package azblob

import (
	"context"
	"crypto/md5"
	"io/ioutil"

	"bytes"
	"strings"
	"time"

	chk "github.com/sfc-gh-cshi/check"
)

func (s *aztestsSuite) TestPutGetPages(c *chk.C) {
	bsu := getBSU()
	container, _ := createNewContainer(c, bsu)
	defer delContainer(c, container)

	blob, _ := createNewPageBlob(c, container)

	pageRange := PageRange{Start: 0, End: 1023}
	putResp, err := blob.UploadPages(context.Background(), 0, getReaderToRandomBytes(1024), PageBlobAccessConditions{}, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)
	c.Assert(putResp.Response().StatusCode, chk.Equals, 201)
	c.Assert(putResp.LastModified().IsZero(), chk.Equals, false)
	c.Assert(putResp.ETag(), chk.Not(chk.Equals), ETagNone)
	c.Assert(putResp.ContentMD5(), chk.Not(chk.Equals), "")
	c.Assert(putResp.BlobSequenceNumber(), chk.Equals, int64(0))
	c.Assert(putResp.RequestID(), chk.Not(chk.Equals), "")
	c.Assert(putResp.Version(), chk.Not(chk.Equals), "")
	c.Assert(putResp.Date().IsZero(), chk.Equals, false)

	pageList, err := blob.GetPageRanges(context.Background(), 0, 1023, BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(pageList.Response().StatusCode, chk.Equals, 200)
	c.Assert(pageList.LastModified().IsZero(), chk.Equals, false)
	c.Assert(pageList.ETag(), chk.Not(chk.Equals), ETagNone)
	c.Assert(pageList.BlobContentLength(), chk.Equals, int64(512*10))
	c.Assert(pageList.RequestID(), chk.Not(chk.Equals), "")
	c.Assert(pageList.Version(), chk.Not(chk.Equals), "")
	c.Assert(pageList.Date().IsZero(), chk.Equals, false)
	c.Assert(pageList.PageRange, chk.HasLen, 1)
	c.Assert(pageList.PageRange[0], chk.DeepEquals, pageRange)
}

func (s *aztestsSuite) TestUploadPagesFromURL(c *chk.C) {
	bsu := getBSU()
	credential, err := getGenericCredential("")
	if err != nil {
		c.Fatal("Invalid credential")
	}
	container, _ := createNewContainer(c, bsu)
	defer delContainer(c, container)

	testSize := 4 * 1024 * 1024 // 4MB
	r, sourceData := getRandomDataAndReader(testSize)
	ctx := context.Background() // Use default Background context
	srcBlob, _ := createNewPageBlobWithSize(c, container, int64(testSize))
	destBlob, _ := createNewPageBlobWithSize(c, container, int64(testSize))

	// Prepare source blob for copy.
	uploadSrcResp1, err := srcBlob.UploadPages(ctx, 0, r, PageBlobAccessConditions{}, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)
	c.Assert(uploadSrcResp1.Response().StatusCode, chk.Equals, 201)

	// Get source blob URL with SAS for UploadPagesFromURL.
	srcBlobParts := NewBlobURLParts(srcBlob.URL())

	srcBlobParts.SAS, err = BlobSASSignatureValues{
		Protocol:      SASProtocolHTTPS,                     // Users MUST use HTTPS (not HTTP)
		ExpiryTime:    time.Now().UTC().Add(48 * time.Hour), // 48-hours before expiration
		ContainerName: srcBlobParts.ContainerName,
		BlobName:      srcBlobParts.BlobName,
		Permissions:   BlobSASPermissions{Read: true}.String(),
	}.NewSASQueryParameters(credential)
	if err != nil {
		c.Fatal(err)
	}

	srcBlobURLWithSAS := srcBlobParts.URL()

	// Upload page from URL.
	pResp1, err := destBlob.UploadPagesFromURL(ctx, srcBlobURLWithSAS, 0, 0, int64(testSize), nil, PageBlobAccessConditions{}, ModifiedAccessConditions{}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)
	c.Assert(pResp1.ETag(), chk.NotNil)
	c.Assert(pResp1.LastModified(), chk.NotNil)
	c.Assert(pResp1.Response().StatusCode, chk.Equals, 201)
	c.Assert(pResp1.ContentMD5(), chk.Not(chk.Equals), "")
	c.Assert(pResp1.RequestID(), chk.Not(chk.Equals), "")
	c.Assert(pResp1.Version(), chk.Not(chk.Equals), "")
	c.Assert(pResp1.Date().IsZero(), chk.Equals, false)

	// Check data integrity through downloading.
	downloadResp, err := destBlob.BlobURL.Download(ctx, 0, CountToEnd, BlobAccessConditions{}, false, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)
	destData, err := ioutil.ReadAll(downloadResp.Body(RetryReaderOptions{}))
	c.Assert(err, chk.IsNil)
	c.Assert(destData, chk.DeepEquals, sourceData)
}

func (s *aztestsSuite) TestUploadPagesFromURLWithMD5(c *chk.C) {
	bsu := getBSU()
	credential, err := getGenericCredential("")
	if err != nil {
		c.Fatal("Invalid credential")
	}
	container, _ := createNewContainer(c, bsu)
	defer delContainer(c, container)

	testSize := 4 * 1024 * 1024 // 4MB
	r, sourceData := getRandomDataAndReader(testSize)
	md5Value := md5.Sum(sourceData)
	ctx := context.Background() // Use default Background context
	srcBlob, _ := createNewPageBlobWithSize(c, container, int64(testSize))
	destBlob, _ := createNewPageBlobWithSize(c, container, int64(testSize))

	// Prepare source blob for copy.
	uploadSrcResp1, err := srcBlob.UploadPages(ctx, 0, r, PageBlobAccessConditions{}, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)
	c.Assert(uploadSrcResp1.Response().StatusCode, chk.Equals, 201)

	// Get source blob URL with SAS for UploadPagesFromURL.
	srcBlobParts := NewBlobURLParts(srcBlob.URL())

	srcBlobParts.SAS, err = BlobSASSignatureValues{
		Protocol:      SASProtocolHTTPS,                     // Users MUST use HTTPS (not HTTP)
		ExpiryTime:    time.Now().UTC().Add(48 * time.Hour), // 48-hours before expiration
		ContainerName: srcBlobParts.ContainerName,
		BlobName:      srcBlobParts.BlobName,
		Permissions:   BlobSASPermissions{Read: true}.String(),
	}.NewSASQueryParameters(credential)
	if err != nil {
		c.Fatal(err)
	}

	srcBlobURLWithSAS := srcBlobParts.URL()

	// Upload page from URL with MD5.
	pResp1, err := destBlob.UploadPagesFromURL(ctx, srcBlobURLWithSAS, 0, 0, int64(testSize), md5Value[:], PageBlobAccessConditions{}, ModifiedAccessConditions{}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)
	c.Assert(pResp1.ETag(), chk.NotNil)
	c.Assert(pResp1.LastModified(), chk.NotNil)
	c.Assert(pResp1.Response().StatusCode, chk.Equals, 201)
	c.Assert(pResp1.RequestID(), chk.Not(chk.Equals), "")
	c.Assert(pResp1.Version(), chk.Not(chk.Equals), "")
	c.Assert(pResp1.Date().IsZero(), chk.Equals, false)
	c.Assert(pResp1.ContentMD5(), chk.DeepEquals, md5Value[:])
	c.Assert(pResp1.BlobSequenceNumber(), chk.Equals, int64(0))

	// Check data integrity through downloading.
	downloadResp, err := destBlob.BlobURL.Download(ctx, 0, CountToEnd, BlobAccessConditions{}, false, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)
	destData, err := ioutil.ReadAll(downloadResp.Body(RetryReaderOptions{}))
	c.Assert(err, chk.IsNil)
	c.Assert(destData, chk.DeepEquals, sourceData)

	// Upload page from URL with bad MD5
	_, badMD5 := getRandomDataAndReader(16)
	_, err = destBlob.UploadPagesFromURL(ctx, srcBlobURLWithSAS, 0, 0, int64(testSize), badMD5[:], PageBlobAccessConditions{}, ModifiedAccessConditions{}, ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeMd5Mismatch)
}

func (s *aztestsSuite) TestClearDiffPages(c *chk.C) {
	bsu := getBSU()
	container, _ := createNewContainer(c, bsu)
	defer delContainer(c, container)

	blob, _ := createNewPageBlob(c, container)
	_, err := blob.UploadPages(context.Background(), 0, getReaderToRandomBytes(2048), PageBlobAccessConditions{}, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	snapshotResp, err := blob.CreateSnapshot(context.Background(), nil, BlobAccessConditions{}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	_, err = blob.UploadPages(context.Background(), 2048, getReaderToRandomBytes(2048), PageBlobAccessConditions{}, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	pageList, err := blob.GetPageRangesDiff(context.Background(), 0, 4096, snapshotResp.Snapshot(), BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(pageList.PageRange, chk.HasLen, 1)
	c.Assert(pageList.PageRange[0].Start, chk.Equals, int64(2048))
	c.Assert(pageList.PageRange[0].End, chk.Equals, int64(4095))

	clearResp, err := blob.ClearPages(context.Background(), 2048, 2048, PageBlobAccessConditions{}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)
	c.Assert(clearResp.Response().StatusCode, chk.Equals, 201)

	pageList, err = blob.GetPageRangesDiff(context.Background(), 0, 4095, snapshotResp.Snapshot(), BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(pageList.PageRange, chk.HasLen, 0)
}

func (s *aztestsSuite) TestIncrementalCopy(c *chk.C) {
	bsu := getBSU()
	container, _ := createNewContainer(c, bsu)
	defer delContainer(c, container)
	_, err := container.SetAccessPolicy(context.Background(), PublicAccessBlob, nil, ContainerAccessConditions{})
	c.Assert(err, chk.IsNil)

	srcBlob, _ := createNewPageBlob(c, container)
	_, err = srcBlob.UploadPages(context.Background(), 0, getReaderToRandomBytes(1024), PageBlobAccessConditions{}, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)
	snapshotResp, err := srcBlob.CreateSnapshot(context.Background(), nil, BlobAccessConditions{}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	dstBlob := container.NewPageBlobURL(generateBlobName())

	resp, err := dstBlob.StartCopyIncremental(context.Background(), srcBlob.URL(), snapshotResp.Snapshot(), BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.Response().StatusCode, chk.Equals, 202)
	c.Assert(resp.LastModified().IsZero(), chk.Equals, false)
	c.Assert(resp.ETag(), chk.Not(chk.Equals), ETagNone)
	c.Assert(resp.RequestID(), chk.Not(chk.Equals), "")
	c.Assert(resp.Version(), chk.Not(chk.Equals), "")
	c.Assert(resp.Date().IsZero(), chk.Equals, false)
	c.Assert(resp.CopyID(), chk.Not(chk.Equals), "")
	c.Assert(resp.CopyStatus(), chk.Equals, CopyStatusPending)

	waitForIncrementalCopy(c, dstBlob, resp)
}

func (s *aztestsSuite) TestResizePageBlob(c *chk.C) {
	bsu := getBSU()
	container, _ := createNewContainer(c, bsu)
	defer delContainer(c, container)

	blob, _ := createNewPageBlob(c, container)
	resp, err := blob.Resize(context.Background(), 2048, BlobAccessConditions{}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.Response().StatusCode, chk.Equals, 200)

	resp, err = blob.Resize(context.Background(), 8192, BlobAccessConditions{}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.Response().StatusCode, chk.Equals, 200)

	resp2, err := blob.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp2.ContentLength(), chk.Equals, int64(8192))
}

func (s *aztestsSuite) TestPageSequenceNumbers(c *chk.C) {
	bsu := getBSU()
	container, _ := createNewContainer(c, bsu)
	blob, _ := createNewPageBlob(c, container)

	defer delContainer(c, container)

	resp, err := blob.UpdateSequenceNumber(context.Background(), SequenceNumberActionIncrement, 0, BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.Response().StatusCode, chk.Equals, 200)

	resp, err = blob.UpdateSequenceNumber(context.Background(), SequenceNumberActionMax, 7, BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.Response().StatusCode, chk.Equals, 200)

	resp, err = blob.UpdateSequenceNumber(context.Background(), SequenceNumberActionUpdate, 11, BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.Response().StatusCode, chk.Equals, 200)
}

func (s *aztestsSuite) TestPutPagesWithMD5(c *chk.C) {
	bsu := getBSU()
	container, _ := createNewContainer(c, bsu)
	defer delContainer(c, container)

	blob, _ := createNewPageBlob(c, container)

	// put page with valid MD5
	readerToBody, body := getRandomDataAndReader(1024)
	md5Value := md5.Sum(body)
	putResp, err := blob.UploadPages(context.Background(), 0, readerToBody, PageBlobAccessConditions{}, md5Value[:], ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)
	c.Assert(putResp.Response().StatusCode, chk.Equals, 201)
	c.Assert(putResp.LastModified().IsZero(), chk.Equals, false)
	c.Assert(putResp.ETag(), chk.Not(chk.Equals), ETagNone)
	c.Assert(putResp.ContentMD5(), chk.DeepEquals, md5Value[:])
	c.Assert(putResp.BlobSequenceNumber(), chk.Equals, int64(0))
	c.Assert(putResp.RequestID(), chk.Not(chk.Equals), "")
	c.Assert(putResp.Version(), chk.Not(chk.Equals), "")
	c.Assert(putResp.Date().IsZero(), chk.Equals, false)

	// put page with bad MD5
	readerToBody, body = getRandomDataAndReader(1024)
	_, badMD5 := getRandomDataAndReader(16)
	putResp, err = blob.UploadPages(context.Background(), 0, readerToBody, PageBlobAccessConditions{}, badMD5[:], ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeMd5Mismatch)
}

func (s *aztestsSuite) TestBlobCreatePageSizeInvalid(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := getPageBlobURL(c, containerURL)

	_, err := blobURL.Create(ctx, 1, 0, BlobHTTPHeaders{}, nil, BlobAccessConditions{}, DefaultPremiumBlobAccessTier, nil, ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeInvalidHeaderValue)
}

func (s *aztestsSuite) TestBlobCreatePageSequenceInvalid(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := getPageBlobURL(c, containerURL)

	_, err := blobURL.Create(ctx, PageBlobPageBytes, -1, BlobHTTPHeaders{}, nil, BlobAccessConditions{}, DefaultPremiumBlobAccessTier, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.Not(chk.IsNil))
}

func (s *aztestsSuite) TestBlobCreatePageMetadataNonEmpty(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := getPageBlobURL(c, containerURL)

	_, err := blobURL.Create(ctx, PageBlobPageBytes, 0, BlobHTTPHeaders{}, basicMetadata, BlobAccessConditions{}, DefaultPremiumBlobAccessTier, nil, ClientProvidedKeyOptions{})

	resp, err := blobURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.NewMetadata(), chk.DeepEquals, basicMetadata)
}

func (s *aztestsSuite) TestBlobCreatePageMetadataEmpty(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := getPageBlobURL(c, containerURL)

	_, err := blobURL.Create(ctx, PageBlobPageBytes, 0, BlobHTTPHeaders{}, Metadata{}, BlobAccessConditions{}, DefaultPremiumBlobAccessTier, nil, ClientProvidedKeyOptions{})

	resp, err := blobURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.NewMetadata(), chk.HasLen, 0)
}

func (s *aztestsSuite) TestBlobCreatePageMetadataInvalid(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := getPageBlobURL(c, containerURL)

	_, err := blobURL.Create(ctx, PageBlobPageBytes, 0, BlobHTTPHeaders{}, Metadata{"In valid1": "bar"}, BlobAccessConditions{}, PremiumPageBlobAccessTierNone, nil, ClientProvidedKeyOptions{})
	c.Assert(strings.Contains(err.Error(), invalidHeaderErrorSubstring), chk.Equals, true)

}

func (s *aztestsSuite) TestBlobCreatePageHTTPHeaders(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := getPageBlobURL(c, containerURL)

	_, err := blobURL.Create(ctx, PageBlobPageBytes, 0, basicHeaders, nil, BlobAccessConditions{}, PremiumPageBlobAccessTierNone, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	resp, err := blobURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)
	h := resp.NewHTTPHeaders()
	c.Assert(h, chk.DeepEquals, basicHeaders)
}

func validatePageBlobPut(c *chk.C, blobURL PageBlobURL) {
	resp, err := blobURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.NewMetadata(), chk.DeepEquals, basicMetadata)
}

func (s *aztestsSuite) TestBlobCreatePageIfModifiedSinceTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL) // Originally created without metadata

	currentTime := getRelativeTimeGMT(-10)

	_, err := blobURL.Create(ctx, PageBlobPageBytes, 0, BlobHTTPHeaders{}, basicMetadata, BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfModifiedSince: currentTime}}, DefaultPremiumBlobAccessTier, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	validatePageBlobPut(c, blobURL)
}

func (s *aztestsSuite) TestBlobCreatePageIfModifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL) // Originally created without metadata

	currentTime := getRelativeTimeGMT(10)

	_, err := blobURL.Create(ctx, PageBlobPageBytes, 0, BlobHTTPHeaders{}, basicMetadata, BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfModifiedSince: currentTime}}, DefaultPremiumBlobAccessTier, nil, ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobCreatePageIfUnmodifiedSinceTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL) // Originally created without metadata

	currentTime := getRelativeTimeGMT(10)

	_, err := blobURL.Create(ctx, PageBlobPageBytes, 0, BlobHTTPHeaders{}, basicMetadata, BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfUnmodifiedSince: currentTime}}, DefaultPremiumBlobAccessTier, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	validatePageBlobPut(c, blobURL)
}

func (s *aztestsSuite) TestBlobCreatePageIfUnmodifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL) // Originally created without metadata

	currentTime := getRelativeTimeGMT(-10)

	_, err := blobURL.Create(ctx, PageBlobPageBytes, 0, BlobHTTPHeaders{}, basicMetadata, BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfUnmodifiedSince: currentTime}}, DefaultPremiumBlobAccessTier, nil, ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobCreatePageIfMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL) // Originally created without metadata

	resp, _ := blobURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})

	_, err := blobURL.Create(ctx, PageBlobPageBytes, 0, BlobHTTPHeaders{}, basicMetadata, BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfMatch: resp.ETag()}}, PremiumPageBlobAccessTierNone, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	validatePageBlobPut(c, blobURL)
}

func (s *aztestsSuite) TestBlobCreatePageIfMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL) // Originally created without metadata

	_, err := blobURL.Create(ctx, PageBlobPageBytes, 0, BlobHTTPHeaders{}, basicMetadata, BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfMatch: ETag("garbage")}}, PremiumPageBlobAccessTierNone, nil, ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobCreatePageIfNoneMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL) // Originally created without metadata

	_, err := blobURL.Create(ctx, PageBlobPageBytes, 0, BlobHTTPHeaders{}, basicMetadata, BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfNoneMatch: ETag("garbage")}}, PremiumPageBlobAccessTierNone, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	validatePageBlobPut(c, blobURL)
}

func (s *aztestsSuite) TestBlobCreatePageIfNoneMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL) // Originally created without metadata

	resp, _ := blobURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})

	_, err := blobURL.Create(ctx, PageBlobPageBytes, 0, BlobHTTPHeaders{}, basicMetadata, BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfNoneMatch: resp.ETag()}}, DefaultPremiumBlobAccessTier, nil, ClientProvidedKeyOptions{})

	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobPutPagesInvalidRange(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	_, err := blobURL.UploadPages(ctx, 0, strings.NewReader(blockBlobDefaultData), PageBlobAccessConditions{}, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.Not(chk.IsNil))
}

func (s *aztestsSuite) TestBlobPutPagesNilBody(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	_, err := blobURL.UploadPages(ctx, 0, nil, PageBlobAccessConditions{}, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.Not(chk.IsNil))
}

func (s *aztestsSuite) TestBlobPutPagesEmptyBody(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	_, err := blobURL.UploadPages(ctx, 0, bytes.NewReader([]byte{}), PageBlobAccessConditions{}, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.Not(chk.IsNil))
}

func (s *aztestsSuite) TestBlobPutPagesNonExistantBlob(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := getPageBlobURL(c, containerURL)

	_, err := blobURL.UploadPages(ctx, 0, getReaderToRandomBytes(PageBlobPageBytes), PageBlobAccessConditions{}, nil, ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeBlobNotFound)
}

func validateUploadPages(c *chk.C, blobURL PageBlobURL) {
	// This will only validate a single put page at 0-PageBlobPageBytes-1
	resp, err := blobURL.GetPageRanges(ctx, 0, 0, BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.PageRange[0], chk.Equals, PageRange{Start: 0, End: PageBlobPageBytes - 1})
}

func (s *aztestsSuite) TestBlobPutPagesIfModifiedSinceTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(-10)

	_, err := blobURL.UploadPages(ctx, 0, getReaderToRandomBytes(PageBlobPageBytes), PageBlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfModifiedSince: currentTime}}, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	validateUploadPages(c, blobURL)
}

func (s *aztestsSuite) TestBlobPutPagesIfModifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	_, err := blobURL.UploadPages(ctx, 0, getReaderToRandomBytes(PageBlobPageBytes), PageBlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfModifiedSince: currentTime}}, nil, ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobPutPagesIfUnmodifiedSinceTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	_, err := blobURL.UploadPages(ctx, 0, getReaderToRandomBytes(PageBlobPageBytes), PageBlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfUnmodifiedSince: currentTime}}, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	validateUploadPages(c, blobURL)
}

func (s *aztestsSuite) TestBlobPutPagesIfUnmodifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(-10)

	_, err := blobURL.UploadPages(ctx, 0, getReaderToRandomBytes(PageBlobPageBytes), PageBlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfUnmodifiedSince: currentTime}}, nil, ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobPutPagesIfMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	resp, _ := blobURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})

	_, err := blobURL.UploadPages(ctx, 0, getReaderToRandomBytes(PageBlobPageBytes), PageBlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfMatch: resp.ETag()}}, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	validateUploadPages(c, blobURL)
}

func (s *aztestsSuite) TestBlobPutPagesIfMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	_, err := blobURL.UploadPages(ctx, 0, getReaderToRandomBytes(PageBlobPageBytes), PageBlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfMatch: ETag("garbage")}}, nil, ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobPutPagesIfNoneMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	_, err := blobURL.UploadPages(ctx, 0, getReaderToRandomBytes(PageBlobPageBytes), PageBlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfNoneMatch: ETag("garbage")}}, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	validateUploadPages(c, blobURL)
}

func (s *aztestsSuite) TestBlobPutPagesIfNoneMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	resp, _ := blobURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})

	_, err := blobURL.UploadPages(ctx, 0, getReaderToRandomBytes(PageBlobPageBytes), PageBlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfNoneMatch: resp.ETag()}}, nil, ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobPutPagesIfSequenceNumberLessThanTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	_, err := blobURL.UploadPages(ctx, 0, getReaderToRandomBytes(PageBlobPageBytes), PageBlobAccessConditions{SequenceNumberAccessConditions: SequenceNumberAccessConditions{IfSequenceNumberLessThan: 10}}, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	validateUploadPages(c, blobURL)
}

func (s *aztestsSuite) TestBlobPutPagesIfSequenceNumberLessThanFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	blobURL.UpdateSequenceNumber(ctx, SequenceNumberActionUpdate, 10, BlobAccessConditions{})
	_, err := blobURL.UploadPages(ctx, 0, getReaderToRandomBytes(PageBlobPageBytes), PageBlobAccessConditions{SequenceNumberAccessConditions: SequenceNumberAccessConditions{IfSequenceNumberLessThan: 1}}, nil, ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeSequenceNumberConditionNotMet)
}

func (s *aztestsSuite) TestBlobPutPagesIfSequenceNumberLessThanNegOne(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	_, err := blobURL.UploadPages(ctx, 0, getReaderToRandomBytes(PageBlobPageBytes), PageBlobAccessConditions{SequenceNumberAccessConditions: SequenceNumberAccessConditions{IfSequenceNumberLessThan: -1}}, nil, ClientProvidedKeyOptions{}) // This will cause the library to set the value of the header to 0
	validateStorageError(c, err, ServiceCodeSequenceNumberConditionNotMet)
}

func (s *aztestsSuite) TestBlobPutPagesIfSequenceNumberLTETrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	blobURL.UpdateSequenceNumber(ctx, SequenceNumberActionUpdate, 1, BlobAccessConditions{})
	_, err := blobURL.UploadPages(ctx, 0, getReaderToRandomBytes(PageBlobPageBytes), PageBlobAccessConditions{SequenceNumberAccessConditions: SequenceNumberAccessConditions{IfSequenceNumberLessThanOrEqual: 1}}, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	validateUploadPages(c, blobURL)
}

func (s *aztestsSuite) TestBlobPutPagesIfSequenceNumberLTEqualFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	blobURL.UpdateSequenceNumber(ctx, SequenceNumberActionUpdate, 10, BlobAccessConditions{})
	_, err := blobURL.UploadPages(ctx, 0, getReaderToRandomBytes(PageBlobPageBytes), PageBlobAccessConditions{SequenceNumberAccessConditions: SequenceNumberAccessConditions{IfSequenceNumberLessThanOrEqual: 1}}, nil, ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeSequenceNumberConditionNotMet)
}

func (s *aztestsSuite) TestBlobPutPagesIfSequenceNumberLTENegOne(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	_, err := blobURL.UploadPages(ctx, 0, getReaderToRandomBytes(PageBlobPageBytes), PageBlobAccessConditions{SequenceNumberAccessConditions: SequenceNumberAccessConditions{IfSequenceNumberLessThanOrEqual: -1}}, nil, ClientProvidedKeyOptions{}) // This will cause the library to set the value of the header to 0
	c.Assert(err, chk.IsNil)

	validateUploadPages(c, blobURL)
}

func (s *aztestsSuite) TestBlobPutPagesIfSequenceNumberEqualTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	blobURL.UpdateSequenceNumber(ctx, SequenceNumberActionUpdate, 1, BlobAccessConditions{})
	_, err := blobURL.UploadPages(ctx, 0, getReaderToRandomBytes(PageBlobPageBytes), PageBlobAccessConditions{SequenceNumberAccessConditions: SequenceNumberAccessConditions{IfSequenceNumberEqual: 1}}, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	validateUploadPages(c, blobURL)
}

func (s *aztestsSuite) TestBlobPutPagesIfSequenceNumberEqualFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	_, err := blobURL.UploadPages(ctx, 0, getReaderToRandomBytes(PageBlobPageBytes), PageBlobAccessConditions{SequenceNumberAccessConditions: SequenceNumberAccessConditions{IfSequenceNumberEqual: 1}}, nil, ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeSequenceNumberConditionNotMet)
}

func (s *aztestsSuite) TestBlobPutPagesIfSequenceNumberEqualNegOne(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	_, err := blobURL.UploadPages(ctx, 0, getReaderToRandomBytes(PageBlobPageBytes), PageBlobAccessConditions{SequenceNumberAccessConditions: SequenceNumberAccessConditions{IfSequenceNumberEqual: -1}}, nil, ClientProvidedKeyOptions{}) // This will cause the library to set the value of the header to 0
	c.Assert(err, chk.IsNil)

	validateUploadPages(c, blobURL)
}

func setupClearPagesTest(c *chk.C) (ContainerURL, PageBlobURL) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	blobURL, _ := createNewPageBlob(c, containerURL)

	_, err := blobURL.UploadPages(ctx, 0, getReaderToRandomBytes(PageBlobPageBytes), PageBlobAccessConditions{}, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	return containerURL, blobURL
}

func validateClearPagesTest(c *chk.C, blobURL PageBlobURL) {
	resp, err := blobURL.GetPageRanges(ctx, 0, 0, BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.PageRange, chk.HasLen, 0)
}

func (s *aztestsSuite) TestBlobClearPagesInvalidRange(c *chk.C) {
	containerURL, blobURL := setupClearPagesTest(c)
	defer deleteContainer(c, containerURL)

	_, err := blobURL.ClearPages(ctx, 0, PageBlobPageBytes+1, PageBlobAccessConditions{}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.Not(chk.IsNil))
}

func (s *aztestsSuite) TestBlobClearPagesIfModifiedSinceTrue(c *chk.C) {
	containerURL, blobURL := setupClearPagesTest(c)
	defer deleteContainer(c, containerURL)

	currentTime := getRelativeTimeGMT(-10)

	_, err := blobURL.ClearPages(ctx, 0, PageBlobPageBytes, PageBlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfModifiedSince: currentTime}}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	validateClearPagesTest(c, blobURL)
}

func (s *aztestsSuite) TestBlobClearPagesIfModifiedSinceFalse(c *chk.C) {
	containerURL, blobURL := setupClearPagesTest(c)
	defer deleteContainer(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	_, err := blobURL.ClearPages(ctx, 0, PageBlobPageBytes, PageBlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfModifiedSince: currentTime}}, ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobClearPagesIfUnmodifiedSinceTrue(c *chk.C) {
	containerURL, blobURL := setupClearPagesTest(c)
	defer deleteContainer(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	_, err := blobURL.ClearPages(ctx, 0, PageBlobPageBytes, PageBlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfUnmodifiedSince: currentTime}}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	validateClearPagesTest(c, blobURL)
}

func (s *aztestsSuite) TestBlobClearPagesIfUnmodifiedSinceFalse(c *chk.C) {
	containerURL, blobURL := setupClearPagesTest(c)
	defer deleteContainer(c, containerURL)

	currentTime := getRelativeTimeGMT(-10)

	_, err := blobURL.ClearPages(ctx, 0, PageBlobPageBytes, PageBlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfUnmodifiedSince: currentTime}}, ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobClearPagesIfMatchTrue(c *chk.C) {
	containerURL, blobURL := setupClearPagesTest(c)
	defer deleteContainer(c, containerURL)

	resp, _ := blobURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})

	_, err := blobURL.ClearPages(ctx, 0, PageBlobPageBytes, PageBlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfMatch: resp.ETag()}}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	validateClearPagesTest(c, blobURL)
}

func (s *aztestsSuite) TestBlobClearPagesIfMatchFalse(c *chk.C) {
	containerURL, blobURL := setupClearPagesTest(c)
	defer deleteContainer(c, containerURL)

	_, err := blobURL.ClearPages(ctx, 0, PageBlobPageBytes, PageBlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfMatch: ETag("garbage")}}, ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobClearPagesIfNoneMatchTrue(c *chk.C) {
	containerURL, blobURL := setupClearPagesTest(c)
	defer deleteContainer(c, containerURL)

	_, err := blobURL.ClearPages(ctx, 0, PageBlobPageBytes, PageBlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfNoneMatch: ETag("garbage")}}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	validateClearPagesTest(c, blobURL)
}

func (s *aztestsSuite) TestBlobClearPagesIfNoneMatchFalse(c *chk.C) {
	containerURL, blobURL := setupClearPagesTest(c)
	defer deleteContainer(c, containerURL)

	resp, _ := blobURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})

	_, err := blobURL.ClearPages(ctx, 0, PageBlobPageBytes, PageBlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfNoneMatch: resp.ETag()}}, ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobClearPagesIfSequenceNumberLessThanTrue(c *chk.C) {
	containerURL, blobURL := setupClearPagesTest(c)
	defer deleteContainer(c, containerURL)

	_, err := blobURL.ClearPages(ctx, 0, PageBlobPageBytes, PageBlobAccessConditions{SequenceNumberAccessConditions: SequenceNumberAccessConditions{IfSequenceNumberLessThan: 10}}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	validateClearPagesTest(c, blobURL)
}

func (s *aztestsSuite) TestBlobClearPagesIfSequenceNumberLessThanFalse(c *chk.C) {
	containerURL, blobURL := setupClearPagesTest(c)
	defer deleteContainer(c, containerURL)

	_, err := blobURL.UpdateSequenceNumber(ctx, SequenceNumberActionUpdate, 10, BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	_, err = blobURL.ClearPages(ctx, 0, PageBlobPageBytes, PageBlobAccessConditions{SequenceNumberAccessConditions: SequenceNumberAccessConditions{IfSequenceNumberLessThan: 1}}, ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeSequenceNumberConditionNotMet)
}

func (s *aztestsSuite) TestBlobClearPagesIfSequenceNumberLessThanNegOne(c *chk.C) {
	containerURL, blobURL := setupClearPagesTest(c)
	defer deleteContainer(c, containerURL)

	_, err := blobURL.ClearPages(ctx, 0, PageBlobPageBytes, PageBlobAccessConditions{SequenceNumberAccessConditions: SequenceNumberAccessConditions{IfSequenceNumberLessThan: -1}}, ClientProvidedKeyOptions{}) // This will cause the library to set the value of the header to 0
	validateStorageError(c, err, ServiceCodeSequenceNumberConditionNotMet)
}

func (s *aztestsSuite) TestBlobClearPagesIfSequenceNumberLTETrue(c *chk.C) {
	containerURL, blobURL := setupClearPagesTest(c)
	defer deleteContainer(c, containerURL)

	_, err := blobURL.ClearPages(ctx, 0, PageBlobPageBytes, PageBlobAccessConditions{SequenceNumberAccessConditions: SequenceNumberAccessConditions{IfSequenceNumberLessThanOrEqual: 10}}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	validateClearPagesTest(c, blobURL)
}

func (s *aztestsSuite) TestBlobClearPagesIfSequenceNumberLTEFalse(c *chk.C) {
	containerURL, blobURL := setupClearPagesTest(c)
	defer deleteContainer(c, containerURL)

	_, err := blobURL.UpdateSequenceNumber(ctx, SequenceNumberActionUpdate, 10, BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	_, err = blobURL.ClearPages(ctx, 0, PageBlobPageBytes, PageBlobAccessConditions{SequenceNumberAccessConditions: SequenceNumberAccessConditions{IfSequenceNumberLessThanOrEqual: 1}}, ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeSequenceNumberConditionNotMet)
}

func (s *aztestsSuite) TestBlobClearPagesIfSequenceNumberLTENegOne(c *chk.C) {
	containerURL, blobURL := setupClearPagesTest(c)
	defer deleteContainer(c, containerURL)

	_, err := blobURL.ClearPages(ctx, 0, PageBlobPageBytes, PageBlobAccessConditions{SequenceNumberAccessConditions: SequenceNumberAccessConditions{IfSequenceNumberLessThanOrEqual: -1}}, ClientProvidedKeyOptions{}) // This will cause the library to set the value of the header to 0
	c.Assert(err, chk.IsNil)

	validateClearPagesTest(c, blobURL)
}

func (s *aztestsSuite) TestBlobClearPagesIfSequenceNumberEqualTrue(c *chk.C) {
	containerURL, blobURL := setupClearPagesTest(c)
	defer deleteContainer(c, containerURL)

	_, err := blobURL.UpdateSequenceNumber(ctx, SequenceNumberActionUpdate, 10, BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	_, err = blobURL.ClearPages(ctx, 0, PageBlobPageBytes, PageBlobAccessConditions{SequenceNumberAccessConditions: SequenceNumberAccessConditions{IfSequenceNumberEqual: 10}}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	validateClearPagesTest(c, blobURL)
}

func (s *aztestsSuite) TestBlobClearPagesIfSequenceNumberEqualFalse(c *chk.C) {
	containerURL, blobURL := setupClearPagesTest(c)
	defer deleteContainer(c, containerURL)

	_, err := blobURL.UpdateSequenceNumber(ctx, SequenceNumberActionUpdate, 10, BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	_, err = blobURL.ClearPages(ctx, 0, PageBlobPageBytes, PageBlobAccessConditions{SequenceNumberAccessConditions: SequenceNumberAccessConditions{IfSequenceNumberEqual: 1}}, ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeSequenceNumberConditionNotMet)
}

func (s *aztestsSuite) TestBlobClearPagesIfSequenceNumberEqualNegOne(c *chk.C) {
	containerURL, blobURL := setupClearPagesTest(c)
	defer deleteContainer(c, containerURL)

	_, err := blobURL.ClearPages(ctx, 0, PageBlobPageBytes, PageBlobAccessConditions{SequenceNumberAccessConditions: SequenceNumberAccessConditions{IfSequenceNumberEqual: -1}}, ClientProvidedKeyOptions{}) // This will cause the library to set the value of the header to 0
	c.Assert(err, chk.IsNil)

	validateClearPagesTest(c, blobURL)
}

func setupGetPageRangesTest(c *chk.C) (containerURL ContainerURL, blobURL PageBlobURL) {
	bsu := getBSU()
	containerURL, _ = createNewContainer(c, bsu)
	blobURL, _ = createNewPageBlob(c, containerURL)

	_, err := blobURL.UploadPages(ctx, 0, getReaderToRandomBytes(PageBlobPageBytes), PageBlobAccessConditions{}, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	return
}

func validateBasicGetPageRanges(c *chk.C, resp *PageList, err error) {
	c.Assert(err, chk.IsNil)
	c.Assert(resp.PageRange, chk.HasLen, 1)
	c.Assert(resp.PageRange[0], chk.Equals, PageRange{Start: 0, End: PageBlobPageBytes - 1})
}

func (s *aztestsSuite) TestBlobGetPageRangesEmptyBlob(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	resp, err := blobURL.GetPageRanges(ctx, 0, 0, BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.PageRange, chk.HasLen, 0)
}

func (s *aztestsSuite) TestBlobGetPageRangesEmptyRange(c *chk.C) {
	containerURL, blobURL := setupGetPageRangesTest(c)
	defer deleteContainer(c, containerURL)

	resp, err := blobURL.GetPageRanges(ctx, 0, 0, BlobAccessConditions{})
	validateBasicGetPageRanges(c, resp, err)
}

func (s *aztestsSuite) TestBlobGetPageRangesInvalidRange(c *chk.C) {
	containerURL, blobURL := setupGetPageRangesTest(c)
	defer deleteContainer(c, containerURL)

	_, err := blobURL.GetPageRanges(ctx, -2, 500, BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
}

func (s *aztestsSuite) TestBlobGetPageRangesNonContiguousRanges(c *chk.C) {
	containerURL, blobURL := setupGetPageRangesTest(c)
	defer deleteContainer(c, containerURL)

	_, err := blobURL.UploadPages(ctx, PageBlobPageBytes*2, getReaderToRandomBytes(PageBlobPageBytes), PageBlobAccessConditions{}, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)
	resp, err := blobURL.GetPageRanges(ctx, 0, 0, BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.PageRange, chk.HasLen, 2)
	c.Assert(resp.PageRange[0], chk.Equals, PageRange{Start: 0, End: PageBlobPageBytes - 1})
	c.Assert(resp.PageRange[1], chk.Equals, PageRange{Start: PageBlobPageBytes * 2, End: (PageBlobPageBytes * 3) - 1})
}
func (s *aztestsSuite) TestblobGetPageRangesNotPageAligned(c *chk.C) {
	containerURL, blobURL := setupGetPageRangesTest(c)
	defer deleteContainer(c, containerURL)

	resp, err := blobURL.GetPageRanges(ctx, 0, 2000, BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	validateBasicGetPageRanges(c, resp, err)
}

func (s *aztestsSuite) TestBlobGetPageRangesSnapshot(c *chk.C) {
	containerURL, blobURL := setupGetPageRangesTest(c)
	defer deleteContainer(c, containerURL)

	resp, _ := blobURL.CreateSnapshot(ctx, nil, BlobAccessConditions{}, ClientProvidedKeyOptions{})
	snapshotURL := blobURL.WithSnapshot(resp.Snapshot())
	resp2, err := snapshotURL.GetPageRanges(ctx, 0, 0, BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	validateBasicGetPageRanges(c, resp2, err)
}

func (s *aztestsSuite) TestBlobGetPageRangesIfModifiedSinceTrue(c *chk.C) {
	containerURL, blobURL := setupGetPageRangesTest(c)
	defer deleteContainer(c, containerURL)

	currentTime := getRelativeTimeGMT(-10)

	resp, err := blobURL.GetPageRanges(ctx, 0, 0,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfModifiedSince: currentTime}})
	validateBasicGetPageRanges(c, resp, err)
}

func (s *aztestsSuite) TestBlobGetPageRangesIfModifiedSinceFalse(c *chk.C) {
	containerURL, blobURL := setupGetPageRangesTest(c)
	defer deleteContainer(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	_, err := blobURL.GetPageRanges(ctx, 0, 0,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfModifiedSince: currentTime}})
	serr := err.(StorageError)
	c.Assert(serr.Response().StatusCode, chk.Equals, 304) // Service Code not returned in the body for a HEAD
}

func (s *aztestsSuite) TestBlobGetPageRangesIfUnmodifiedSinceTrue(c *chk.C) {
	containerURL, blobURL := setupGetPageRangesTest(c)
	defer deleteContainer(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	resp, err := blobURL.GetPageRanges(ctx, 0, 0,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	validateBasicGetPageRanges(c, resp, err)
}

func (s *aztestsSuite) TestBlobGetPageRangesIfUnmodifiedSinceFalse(c *chk.C) {
	containerURL, blobURL := setupGetPageRangesTest(c)
	defer deleteContainer(c, containerURL)

	currentTime := getRelativeTimeGMT(-10)

	_, err := blobURL.GetPageRanges(ctx, 0, 0,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobGetPageRangesIfMatchTrue(c *chk.C) {
	containerURL, blobURL := setupGetPageRangesTest(c)
	defer deleteContainer(c, containerURL)

	resp, _ := blobURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})

	resp2, err := blobURL.GetPageRanges(ctx, 0, 0,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfMatch: resp.ETag()}})
	validateBasicGetPageRanges(c, resp2, err)
}

func (s *aztestsSuite) TestBlobGetPageRangesIfMatchFalse(c *chk.C) {
	containerURL, blobURL := setupGetPageRangesTest(c)
	defer deleteContainer(c, containerURL)

	_, err := blobURL.GetPageRanges(ctx, 0, 0,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfMatch: ETag("garbage")}})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobGetPageRangesIfNoneMatchTrue(c *chk.C) {
	containerURL, blobURL := setupGetPageRangesTest(c)
	defer deleteContainer(c, containerURL)

	resp, err := blobURL.GetPageRanges(ctx, 0, 0,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfNoneMatch: ETag("garbage")}})
	validateBasicGetPageRanges(c, resp, err)
}

func (s *aztestsSuite) TestBlobGetPageRangesIfNoneMatchFalse(c *chk.C) {
	containerURL, blobURL := setupGetPageRangesTest(c)
	defer deleteContainer(c, containerURL)

	resp, _ := blobURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})

	_, err := blobURL.GetPageRanges(ctx, 0, 0,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfNoneMatch: resp.ETag()}})
	serr := err.(StorageError)
	c.Assert(serr.Response().StatusCode, chk.Equals, 304) // Service Code not returned in the body for a HEAD
}

func setupDiffPageRangesTest(c *chk.C) (containerURL ContainerURL, blobURL PageBlobURL, snapshot string) {
	bsu := getBSU()
	containerURL, _ = createNewContainer(c, bsu)
	blobURL, _ = createNewPageBlob(c, containerURL)

	_, err := blobURL.UploadPages(ctx, 0, getReaderToRandomBytes(PageBlobPageBytes), PageBlobAccessConditions{}, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	resp, err := blobURL.CreateSnapshot(ctx, nil, BlobAccessConditions{}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)
	snapshot = resp.Snapshot()

	_, err = blobURL.UploadPages(ctx, 0, getReaderToRandomBytes(PageBlobPageBytes), PageBlobAccessConditions{}, nil, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil) // This ensures there is a diff on the first page
	return
}

func validateDiffPageRanges(c *chk.C, resp *PageList, err error) {
	c.Assert(err, chk.IsNil)
	c.Assert(resp.PageRange, chk.HasLen, 1)
	c.Assert(resp.PageRange[0].Start, chk.Equals, int64(0))
	c.Assert(resp.PageRange[0].End, chk.Equals, int64(PageBlobPageBytes-1))
}

func (s *aztestsSuite) TestBlobDiffPageRangesNonExistantSnapshot(c *chk.C) {
	containerURL, blobURL, snapshot := setupDiffPageRangesTest(c)
	defer deleteContainer(c, containerURL)

	snapshotTime, _ := time.Parse(SnapshotTimeFormat, snapshot)
	snapshotTime = snapshotTime.Add(time.Minute)
	_, err := blobURL.GetPageRangesDiff(ctx, 0, 0, snapshotTime.Format(SnapshotTimeFormat), BlobAccessConditions{})
	validateStorageError(c, err, ServiceCodePreviousSnapshotNotFound)
}

func (s *aztestsSuite) TestBlobDiffPageRangeInvalidRange(c *chk.C) {
	containerURL, blobURL, snapshot := setupDiffPageRangesTest(c)
	defer deleteContainer(c, containerURL)
	_, err := blobURL.GetPageRangesDiff(ctx, -22, 14, snapshot, BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
}

func (s *aztestsSuite) TestBlobDiffPageRangeIfModifiedSinceTrue(c *chk.C) {
	containerURL, blobURL, snapshot := setupDiffPageRangesTest(c)
	defer deleteContainer(c, containerURL)

	currentTime := getRelativeTimeGMT(-10)

	resp, err := blobURL.GetPageRangesDiff(ctx, 0, 0, snapshot,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfModifiedSince: currentTime}})
	validateDiffPageRanges(c, resp, err)
}

func (s *aztestsSuite) TestBlobDiffPageRangeIfModifiedSinceFalse(c *chk.C) {
	containerURL, blobURL, snapshot := setupDiffPageRangesTest(c)
	defer deleteContainer(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	_, err := blobURL.GetPageRangesDiff(ctx, 0, 0, snapshot,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfModifiedSince: currentTime}})
	serr := err.(StorageError)
	c.Assert(serr.Response().StatusCode, chk.Equals, 304) // Service Code not returned in the body for a HEAD
}

func (s *aztestsSuite) TestBlobDiffPageRangeIfUnmodifiedSinceTrue(c *chk.C) {
	containerURL, blobURL, snapshot := setupDiffPageRangesTest(c)
	defer deleteContainer(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	resp, err := blobURL.GetPageRangesDiff(ctx, 0, 0, snapshot,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	validateDiffPageRanges(c, resp, err)
}

func (s *aztestsSuite) TestBlobDiffPageRangeIfUnmodifiedSinceFalse(c *chk.C) {
	containerURL, blobURL, snapshot := setupDiffPageRangesTest(c)
	defer deleteContainer(c, containerURL)

	currentTime := getRelativeTimeGMT(-10)

	_, err := blobURL.GetPageRangesDiff(ctx, 0, 0, snapshot,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobDiffPageRangeIfMatchTrue(c *chk.C) {
	containerURL, blobURL, snapshot := setupDiffPageRangesTest(c)
	defer deleteContainer(c, containerURL)

	resp, _ := blobURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})

	resp2, err := blobURL.GetPageRangesDiff(ctx, 0, 0, snapshot,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfMatch: resp.ETag()}})
	validateDiffPageRanges(c, resp2, err)
}

func (s *aztestsSuite) TestBlobDiffPageRangeIfMatchFalse(c *chk.C) {
	containerURL, blobURL, snapshot := setupDiffPageRangesTest(c)
	defer deleteContainer(c, containerURL)

	_, err := blobURL.GetPageRangesDiff(ctx, 0, 0, snapshot,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfMatch: ETag("garbage")}})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobDiffPageRangeIfNoneMatchTrue(c *chk.C) {
	containerURL, blobURL, snapshot := setupDiffPageRangesTest(c)
	defer deleteContainer(c, containerURL)

	resp, err := blobURL.GetPageRangesDiff(ctx, 0, 0, snapshot,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfNoneMatch: ETag("garbage")}})
	validateDiffPageRanges(c, resp, err)
}

func (s *aztestsSuite) TestBlobDiffPageRangeIfNoneMatchFalse(c *chk.C) {
	containerURL, blobURL, snapshot := setupDiffPageRangesTest(c)
	defer deleteContainer(c, containerURL)

	resp, _ := blobURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})

	_, err := blobURL.GetPageRangesDiff(ctx, 0, 0, snapshot,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfNoneMatch: resp.ETag()}})
	serr := err.(StorageError)
	c.Assert(serr.Response().StatusCode, chk.Equals, 304) // Service Code not returned in the body for a HEAD
}

func (s *aztestsSuite) TestBlobResizeZero(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	// The default blob is created with size > 0, so this should actually update
	_, err := blobURL.Resize(ctx, 0, BlobAccessConditions{}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	resp, err := blobURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.ContentLength(), chk.Equals, int64(0))
}

func (s *aztestsSuite) TestBlobResizeInvalidSizeNegative(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	_, err := blobURL.Resize(ctx, -4, BlobAccessConditions{}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.Not(chk.IsNil))
}

func (s *aztestsSuite) TestBlobResizeInvalidSizeMisaligned(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	_, err := blobURL.Resize(ctx, 12, BlobAccessConditions{}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.Not(chk.IsNil))
}

func validateResize(c *chk.C, blobURL PageBlobURL) {
	resp, _ := blobURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})
	c.Assert(resp.ContentLength(), chk.Equals, int64(PageBlobPageBytes))
}

func (s *aztestsSuite) TestBlobResizeIfModifiedSinceTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(-10)

	_, err := blobURL.Resize(ctx, PageBlobPageBytes, BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfModifiedSince: currentTime}}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	validateResize(c, blobURL)
}

func (s *aztestsSuite) TestBlobResizeIfModifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	_, err := blobURL.Resize(ctx, PageBlobPageBytes, BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfModifiedSince: currentTime}}, ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobResizeIfUnmodifiedSinceTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	_, err := blobURL.Resize(ctx, PageBlobPageBytes, BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfUnmodifiedSince: currentTime}}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	validateResize(c, blobURL)
}

func (s *aztestsSuite) TestBlobResizeIfUnmodifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(-10)

	_, err := blobURL.Resize(ctx, PageBlobPageBytes, BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfUnmodifiedSince: currentTime}}, ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobResizeIfMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	resp, _ := blobURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})

	_, err := blobURL.Resize(ctx, PageBlobPageBytes, BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfMatch: resp.ETag()}}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	validateResize(c, blobURL)
}

func (s *aztestsSuite) TestBlobResizeIfMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	_, err := blobURL.Resize(ctx, PageBlobPageBytes, BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfMatch: ETag("garbage")}}, ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobResizeIfNoneMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	_, err := blobURL.Resize(ctx, PageBlobPageBytes, BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfNoneMatch: ETag("garbage")}}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)

	validateResize(c, blobURL)
}

func (s *aztestsSuite) TestBlobResizeIfNoneMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	resp, _ := blobURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})

	_, err := blobURL.Resize(ctx, PageBlobPageBytes, BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfNoneMatch: resp.ETag()}}, ClientProvidedKeyOptions{})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobSetSequenceNumberActionTypeInvalid(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	_, err := blobURL.UpdateSequenceNumber(ctx, SequenceNumberActionType("garbage"), 1, BlobAccessConditions{})
	validateStorageError(c, err, ServiceCodeInvalidHeaderValue)
}

func (s *aztestsSuite) TestBlobSetSequenceNumberSequenceNumberInvalid(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	defer func() { // Invalid sequence number should panic
		recover()
	}()

	blobURL.UpdateSequenceNumber(ctx, SequenceNumberActionUpdate, -1, BlobAccessConditions{})
}

func validateSequenceNumberSet(c *chk.C, blobURL PageBlobURL) {
	resp, err := blobURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)
	c.Assert(resp.BlobSequenceNumber(), chk.Equals, int64(1))
}

func (s *aztestsSuite) TestBlobSetSequenceNumberIfModifiedSinceTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(-10)

	_, err := blobURL.UpdateSequenceNumber(ctx, SequenceNumberActionIncrement, 0,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfModifiedSince: currentTime}})
	c.Assert(err, chk.IsNil)

	validateSequenceNumberSet(c, blobURL)
}

func (s *aztestsSuite) TestBlobSetSequenceNumberIfModifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	_, err := blobURL.UpdateSequenceNumber(ctx, SequenceNumberActionIncrement, 0,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfModifiedSince: currentTime}})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobSetSequenceNumberIfUnmodifiedSinceTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(10)

	_, err := blobURL.UpdateSequenceNumber(ctx, SequenceNumberActionIncrement, 0,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	c.Assert(err, chk.IsNil)

	validateSequenceNumberSet(c, blobURL)
}

func (s *aztestsSuite) TestBlobSetSequenceNumberIfUnmodifiedSinceFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	currentTime := getRelativeTimeGMT(-10)

	_, err := blobURL.UpdateSequenceNumber(ctx, SequenceNumberActionIncrement, 0,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobSetSequenceNumberIfMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	resp, _ := blobURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})

	_, err := blobURL.UpdateSequenceNumber(ctx, SequenceNumberActionIncrement, 0,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfMatch: resp.ETag()}})
	c.Assert(err, chk.IsNil)

	validateSequenceNumberSet(c, blobURL)
}

func (s *aztestsSuite) TestBlobSetSequenceNumberIfMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	_, err := blobURL.UpdateSequenceNumber(ctx, SequenceNumberActionIncrement, 0,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfMatch: ETag("garbage")}})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobSetSequenceNumberIfNoneMatchTrue(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	_, err := blobURL.UpdateSequenceNumber(ctx, SequenceNumberActionIncrement, 0,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfNoneMatch: ETag("garbage")}})
	c.Assert(err, chk.IsNil)

	validateSequenceNumberSet(c, blobURL)
}

func (s *aztestsSuite) TestBlobSetSequenceNumberIfNoneMatchFalse(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)

	resp, _ := blobURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})

	_, err := blobURL.UpdateSequenceNumber(ctx, SequenceNumberActionIncrement, 0,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfNoneMatch: resp.ETag()}})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func waitForIncrementalCopy(c *chk.C, copyBlobURL PageBlobURL, blobCopyResponse *PageBlobCopyIncrementalResponse) string {
	status := blobCopyResponse.CopyStatus()
	var getPropertiesAndMetadataResult *BlobGetPropertiesResponse
	// Wait for the copy to finish
	start := time.Now()
	for status != CopyStatusSuccess {
		getPropertiesAndMetadataResult, _ = copyBlobURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})
		status = getPropertiesAndMetadataResult.CopyStatus()
		currentTime := time.Now()
		if currentTime.Sub(start) >= time.Minute {
			c.Fail()
		}
	}
	return getPropertiesAndMetadataResult.DestinationSnapshot()
}

func setupStartIncrementalCopyTest(c *chk.C) (containerURL ContainerURL, blobURL PageBlobURL, copyBlobURL PageBlobURL, snapshot string) {
	bsu := getBSU()
	containerURL, _ = createNewContainer(c, bsu)
	containerURL.SetAccessPolicy(ctx, PublicAccessBlob, nil, ContainerAccessConditions{})
	blobURL, _ = createNewPageBlob(c, containerURL)
	resp, _ := blobURL.CreateSnapshot(ctx, nil, BlobAccessConditions{}, ClientProvidedKeyOptions{})
	copyBlobURL, _ = getPageBlobURL(c, containerURL)

	// Must create the incremental copy blob so that the access conditions work on it
	resp2, err := copyBlobURL.StartCopyIncremental(ctx, blobURL.URL(), resp.Snapshot(), BlobAccessConditions{})
	c.Assert(err, chk.IsNil)
	waitForIncrementalCopy(c, copyBlobURL, resp2)

	resp, _ = blobURL.CreateSnapshot(ctx, nil, BlobAccessConditions{}, ClientProvidedKeyOptions{}) // Take a new snapshot so the next copy will succeed
	snapshot = resp.Snapshot()
	return
}

func validateIncrementalCopy(c *chk.C, copyBlobURL PageBlobURL, resp *PageBlobCopyIncrementalResponse) {
	t := waitForIncrementalCopy(c, copyBlobURL, resp)

	// If we can access the snapshot without error, we are satisfied that it was created as a result of the copy
	copySnapshotURL := copyBlobURL.WithSnapshot(t)
	_, err := copySnapshotURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})
	c.Assert(err, chk.IsNil)
}

func (s *aztestsSuite) TestBlobStartIncrementalCopySnapshotNotExist(c *chk.C) {
	bsu := getBSU()
	containerURL, _ := createNewContainer(c, bsu)
	defer deleteContainer(c, containerURL)
	blobURL, _ := createNewPageBlob(c, containerURL)
	copyBlobURL, _ := getPageBlobURL(c, containerURL)

	snapshot := time.Now().UTC().Format(SnapshotTimeFormat)
	_, err := copyBlobURL.StartCopyIncremental(ctx, blobURL.URL(), snapshot, BlobAccessConditions{})
	validateStorageError(c, err, ServiceCodeCannotVerifyCopySource)
}

func (s *aztestsSuite) TestBlobStartIncrementalCopyIfModifiedSinceTrue(c *chk.C) {
	containerURL, blobURL, copyBlobURL, snapshot := setupStartIncrementalCopyTest(c)

	defer deleteContainer(c, containerURL)

	currentTime := getRelativeTimeGMT(-20)

	resp, err := copyBlobURL.StartCopyIncremental(ctx, blobURL.URL(), snapshot,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfModifiedSince: currentTime}})
	c.Assert(err, chk.IsNil)

	validateIncrementalCopy(c, copyBlobURL, resp)
}

func (s *aztestsSuite) TestBlobStartIncrementalCopyIfModifiedSinceFalse(c *chk.C) {
	containerURL, blobURL, copyBlobURL, snapshot := setupStartIncrementalCopyTest(c)

	defer deleteContainer(c, containerURL)

	currentTime := getRelativeTimeGMT(20)

	_, err := copyBlobURL.StartCopyIncremental(ctx, blobURL.URL(), snapshot,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfModifiedSince: currentTime}})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobStartIncrementalCopyIfUnmodifiedSinceTrue(c *chk.C) {
	containerURL, blobURL, copyBlobURL, snapshot := setupStartIncrementalCopyTest(c)

	defer deleteContainer(c, containerURL)

	currentTime := getRelativeTimeGMT(20)

	resp, err := copyBlobURL.StartCopyIncremental(ctx, blobURL.URL(), snapshot,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	c.Assert(err, chk.IsNil)

	validateIncrementalCopy(c, copyBlobURL, resp)
}

func (s *aztestsSuite) TestBlobStartIncrementalCopyIfUnmodifiedSinceFalse(c *chk.C) {
	containerURL, blobURL, copyBlobURL, snapshot := setupStartIncrementalCopyTest(c)

	defer deleteContainer(c, containerURL)

	currentTime := getRelativeTimeGMT(-20)

	_, err := copyBlobURL.StartCopyIncremental(ctx, blobURL.URL(), snapshot,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfUnmodifiedSince: currentTime}})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}

func (s *aztestsSuite) TestBlobStartIncrementalCopyIfMatchTrue(c *chk.C) {
	containerURL, blobURL, copyBlobURL, snapshot := setupStartIncrementalCopyTest(c)

	defer deleteContainer(c, containerURL)

	resp, _ := copyBlobURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})

	resp2, err := copyBlobURL.StartCopyIncremental(ctx, blobURL.URL(), snapshot,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfMatch: resp.ETag()}})
	c.Assert(err, chk.IsNil)

	validateIncrementalCopy(c, copyBlobURL, resp2)
}

func (s *aztestsSuite) TestBlobStartIncrementalCopyIfMatchFalse(c *chk.C) {
	containerURL, blobURL, copyBlobURL, snapshot := setupStartIncrementalCopyTest(c)

	defer deleteContainer(c, containerURL)

	_, err := copyBlobURL.StartCopyIncremental(ctx, blobURL.URL(), snapshot,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfMatch: ETag("garbage")}})
	validateStorageError(c, err, ServiceCodeTargetConditionNotMet)
}

func (s *aztestsSuite) TestBlobStartIncrementalCopyIfNoneMatchTrue(c *chk.C) {
	containerURL, blobURL, copyBlobURL, snapshot := setupStartIncrementalCopyTest(c)

	defer deleteContainer(c, containerURL)

	resp, err := copyBlobURL.StartCopyIncremental(ctx, blobURL.URL(), snapshot,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfNoneMatch: ETag("garbage")}})
	c.Assert(err, chk.IsNil)

	validateIncrementalCopy(c, copyBlobURL, resp)
}

func (s *aztestsSuite) TestBlobStartIncrementalCopyIfNoneMatchFalse(c *chk.C) {
	containerURL, blobURL, copyBlobURL, snapshot := setupStartIncrementalCopyTest(c)

	defer deleteContainer(c, containerURL)

	resp, _ := copyBlobURL.GetProperties(ctx, BlobAccessConditions{}, ClientProvidedKeyOptions{})

	_, err := copyBlobURL.StartCopyIncremental(ctx, blobURL.URL(), snapshot,
		BlobAccessConditions{ModifiedAccessConditions: ModifiedAccessConditions{IfNoneMatch: resp.ETag()}})
	validateStorageError(c, err, ServiceCodeConditionNotMet)
}
